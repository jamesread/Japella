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
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/intconv"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/media"
	"github.com/jamesread/japella/internal/medialimits"
)

// GetMediaSuitability checks a library file against the limits of each
// network the caller has an active social account on.
func (s *ControlApi) GetMediaSuitability(ctx context.Context, req *connect.Request[controlv1.GetMediaSuitabilityRequest]) (*connect.Response[controlv1.GetMediaSuitabilityResponse], error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, err
	}

	info, err := media.Inspect(req.Msg.GetFilename())
	if errors.Is(err, os.ErrNotExist) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("media file not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read media file"))
	}

	return connect.NewResponse(&controlv1.GetMediaSuitabilityResponse{
		SizeBytes: intconv.ToUint64(info.SizeBytes),
		Width:     intconv.ToUint32(info.Width),
		Height:    intconv.ToUint32(info.Height),
		Networks:  s.mediaSuitability(s.userMediaNetworks(au), info),
	}), nil
}

// mediaNetwork is a network with known media limits that the user posts to.
type mediaNetwork struct {
	protocol string
	icon     string
	limits   medialimits.Limits
}

func (s *ControlApi) accountsForMediaNetworks(au *authentication.AuthenticatedUser) []*db.SocialAccount {
	if canViewAllSocialAccounts(au) {
		return s.DB.SelectSocialAccounts(true)
	}
	return s.DB.SelectSocialAccountsForUser(au.User.ID, true)
}

func (s *ControlApi) mediaNetworkFromAccount(sa *db.SocialAccount, seen map[string]bool) (mediaNetwork, bool) {
	if seen[sa.Connector] {
		return mediaNetwork{}, false
	}
	seen[sa.Connector] = true
	limits, ok := medialimits.ForProtocol(sa.Connector)
	if !ok {
		return mediaNetwork{}, false
	}
	n := mediaNetwork{protocol: sa.Connector, icon: "mdi:question-mark-circle", limits: limits}
	if s.cc != nil {
		if svc := s.cc.Get(sa.Connector); svc != nil {
			n.icon = svc.GetIcon()
		}
	}
	return n, true
}

// userMediaNetworks returns the networks, with known limits, of the user's
// active social accounts, in name order.
func (s *ControlApi) userMediaNetworks(au *authentication.AuthenticatedUser) []mediaNetwork {
	networks := make([]mediaNetwork, 0)
	seen := map[string]bool{}
	for _, sa := range s.accountsForMediaNetworks(au) {
		if n, ok := s.mediaNetworkFromAccount(sa, seen); ok {
			networks = append(networks, n)
		}
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

func runMediaOptimise(au *authentication.AuthenticatedUser, s *ControlApi, filename, mode string) (*media.OptimiseResult, string, error) {
	switch mode {
	case "", "lossless":
		result, err := media.Optimise(filename)
		return result, media.KindLossless, err
	case "posting":
		result, err := media.OptimiseForPosting(filename, postingTarget(s.userMediaNetworks(au)))
		return result, media.KindPosting, err
	default:
		return nil, "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("mode must be \"lossless\" or \"posting\""))
	}
}

func mapOptimiseMediaError(err error) error {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("media file not found"))
	case errors.Is(err, media.ErrNotOptimisable):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case err != nil:
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return nil
}

func (s *ControlApi) optimisedMediaItem(au *authentication.AuthenticatedUser, originalFilename string, result *media.OptimiseResult, kind string) *controlv1.MediaItem {
	if result.Filename == "" {
		return nil
	}
	item := &controlv1.MediaItem{
		Filename:      result.Filename,
		Url:           "/media/files/" + result.Filename,
		SizeBytes:     intconv.ToUint64(result.OptimisedBytes),
		OptimisedOf:   originalFilename,
		OptimisedKind: kind,
	}
	if info, err := media.Inspect(result.Filename); err == nil {
		item.Width, item.Height = intconv.ToUint32(info.Width), intconv.ToUint32(info.Height)
		item.Suitability = s.mediaSuitability(s.userMediaNetworks(au), info)
	}
	return item
}

// OptimiseMedia writes a lossless, smaller copy of a library image beside
// the original.
func (s *ControlApi) OptimiseMedia(ctx context.Context, req *connect.Request[controlv1.OptimiseMediaRequest]) (*connect.Response[controlv1.OptimiseMediaResponse], error) {
	au, err := s.requireAuthenticatedAccount(ctx)
	if err != nil {
		return nil, err
	}
	result, kind, err := runMediaOptimise(au, s, req.Msg.GetFilename(), req.Msg.GetMode())
	if mapErr := mapOptimiseMediaError(err); mapErr != nil {
		return nil, mapErr
	}
	res := &controlv1.OptimiseMediaResponse{
		OriginalSizeBytes:  intconv.ToUint64(result.OriginalBytes),
		OptimisedSizeBytes: intconv.ToUint64(result.OptimisedBytes),
		Message:            result.Note,
		Item:               s.optimisedMediaItem(au, req.Msg.GetFilename(), result, kind),
	}
	return connect.NewResponse(res), nil
}

func tighterLimit(cur, v int64) int64 {
	if v > 0 && (cur == 0 || v < cur) {
		return v
	}
	return cur
}

func postingSideLimit(l medialimits.Limits) int64 {
	side := tighterLimit(0, int64(l.ResizeLongestSide))
	side = tighterLimit(side, int64(min(l.MaxWidth, l.MaxHeight)))
	if l.MaxPixels > 0 {
		side = tighterLimit(side, int64(math.Sqrt(float64(l.MaxPixels))))
	}
	return side
}

// postingTarget is the strictest limit among networks: the smallest image
// size limit and the smallest size any of them shows images at.
func postingTarget(networks []mediaNetwork) media.PostingTarget {
	t := media.PostingTarget{}
	side := int64(0)
	for _, n := range networks {
		t.MaxBytes = tighterLimit(t.MaxBytes, n.limits.ImageMaxBytes)
		side = tighterLimit(side, postingSideLimit(n.limits))
	}
	t.MaxSide = int(side)
	return t
}
