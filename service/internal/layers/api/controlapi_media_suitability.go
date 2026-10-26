package api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/media"
	"github.com/jamesread/japella/internal/medialimits"
	"github.com/jamesread/japella/internal/rbac"
)

// GetMediaSuitability checks a library file against the limits of each
// network the caller has an active social account on.
func (s *ControlApi) GetMediaSuitability(ctx context.Context, req *connect.Request[controlv1.GetMediaSuitabilityRequest]) (*connect.Response[controlv1.GetMediaSuitabilityResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	info, err := media.Inspect(req.Msg.GetFilename())
	if errors.Is(err, os.ErrNotExist) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("media file not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read media file"))
	}

	return connect.NewResponse(&controlv1.GetMediaSuitabilityResponse{
		SizeBytes: uint64(info.SizeBytes),
		Width:     uint32(info.Width),
		Height:    uint32(info.Height),
		Networks:  s.mediaSuitability(s.userMediaNetworks(au), info),
	}), nil
}

// mediaNetwork is a network with known media limits that the user posts to.
type mediaNetwork struct {
	protocol string
	icon     string
	limits   medialimits.Limits
}

// userMediaNetworks returns the networks, with known limits, of the user's
// active social accounts, in name order.
func (s *ControlApi) userMediaNetworks(au *authentication.AuthenticatedUser) []mediaNetwork {
	accounts := s.DB.SelectSocialAccountsForUser(au.User.ID, true)
	if au.HasPermission(rbac.PermissionSocialAccountsViewAll) || (au.RBAC != nil && au.RBAC.IsSuperuser) {
		accounts = s.DB.SelectSocialAccounts(true)
	}

	networks := make([]mediaNetwork, 0)
	seen := map[string]bool{}
	for _, sa := range accounts {
		if seen[sa.Connector] {
			continue
		}
		seen[sa.Connector] = true
		limits, ok := medialimits.ForProtocol(sa.Connector)
		if !ok {
			continue
		}
		n := mediaNetwork{protocol: sa.Connector, icon: "mdi:question-mark-circle", limits: limits}
		if s.cc != nil {
			if svc := s.cc.Get(sa.Connector); svc != nil {
				n.icon = svc.GetIcon()
			}
		}
		networks = append(networks, n)
	}
	sort.Slice(networks, func(i, j int) bool { return networks[i].limits.Name < networks[j].limits.Name })
	return networks
}

// mediaSuitability checks one file against each network.
func (s *ControlApi) mediaSuitability(networks []mediaNetwork, info *media.Info) []*controlv1.MediaSuitability {
	out := make([]*controlv1.MediaSuitability, 0, len(networks))
	for _, n := range networks {
		verdict, checks := medialimits.Evaluate(n.limits, medialimits.Media{
			Filename:  info.Filename,
			SizeBytes: info.SizeBytes,
			Width:     info.Width,
			Height:    info.Height,
		})
		ms := &controlv1.MediaSuitability{Protocol: n.protocol, Name: n.limits.Name, Icon: n.icon, Verdict: verdict}
		for _, c := range checks {
			ms.Checks = append(ms.Checks, &controlv1.MediaSuitabilityCheck{Level: c.Level, Message: c.Message})
		}
		out = append(out, ms)
	}
	return out
}

// OptimiseMedia writes a lossless, smaller copy of a library image beside
// the original.
func (s *ControlApi) OptimiseMedia(ctx context.Context, req *connect.Request[controlv1.OptimiseMediaRequest]) (*connect.Response[controlv1.OptimiseMediaResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil || au.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	var result *media.OptimiseResult
	var err error
	kind := media.KindLossless
	switch req.Msg.GetMode() {
	case "", "lossless":
		result, err = media.Optimise(req.Msg.GetFilename())
	case "posting":
		kind = media.KindPosting
		result, err = media.OptimiseForPosting(req.Msg.GetFilename(), postingTarget(s.userMediaNetworks(au)))
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("mode must be \"lossless\" or \"posting\""))
	}
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("media file not found"))
	case errors.Is(err, media.ErrNotOptimisable):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	res := &controlv1.OptimiseMediaResponse{
		OriginalSizeBytes:  uint64(result.OriginalBytes),
		OptimisedSizeBytes: uint64(result.OptimisedBytes),
		Message:            result.Note,
	}
	if result.Filename != "" {
		item := &controlv1.MediaItem{
			Filename:      result.Filename,
			Url:           "/media/files/" + result.Filename,
			SizeBytes:     uint64(result.OptimisedBytes),
			OptimisedOf:   req.Msg.GetFilename(),
			OptimisedKind: kind,
		}
		if info, err := media.Inspect(result.Filename); err == nil {
			item.Width, item.Height = uint32(info.Width), uint32(info.Height)
			item.Suitability = s.mediaSuitability(s.userMediaNetworks(au), info)
		}
		res.Item = item
	}
	return connect.NewResponse(res), nil
}

// postingTarget is the strictest limit among networks: the smallest image
// size limit and the smallest size any of them shows images at.
func postingTarget(networks []mediaNetwork) media.PostingTarget {
	t := media.PostingTarget{}
	tighter := func(cur, v int64) int64 {
		if v > 0 && (cur == 0 || v < cur) {
			return v
		}
		return cur
	}
	side := int64(0)
	for _, n := range networks {
		l := n.limits
		t.MaxBytes = tighter(t.MaxBytes, l.ImageMaxBytes)
		side = tighter(side, int64(l.ResizeLongestSide))
		side = tighter(side, int64(min(l.MaxWidth, l.MaxHeight)))
		if l.MaxPixels > 0 {
			side = tighter(side, int64(math.Sqrt(float64(l.MaxPixels))))
		}
	}
	t.MaxSide = int(side)
	return t
}
