package api

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/postreplies"
)

func (s *ControlApi) connectorSupportsReplies(connectorName string) bool {
	if s.cc == nil {
		return false
	}
	_, ok := s.cc.Get(connectorName).(connector.ConnectorWithPostReplies)
	return ok
}

func replyNodesToProto(nodes []*postreplies.Node) []*controlv1.PostReply {
	out := make([]*controlv1.PostReply, 0, len(nodes))
	for _, n := range nodes {
		r := n.Reply
		pr := &controlv1.PostReply{
			RemoteId:        r.RemoteID,
			RemoteUrl:       r.RemoteURL,
			AuthorName:      r.AuthorName,
			AuthorHandle:    r.AuthorHandle,
			AuthorAvatarUrl: r.AuthorAvatarURL,
			Content:         r.Content,
			Replies:         replyNodesToProto(n.Replies),
		}
		if r.PostedAt.Valid {
			pr.PostedAt = r.PostedAt.Time.UTC().Format(time.RFC3339)
		}
		out = append(out, pr)
	}
	return out
}

// GetPostReplies returns the stored reply thread under a published post, as
// last fetched by the post_engagement job.
func (s *ControlApi) GetPostReplies(ctx context.Context, req *connect.Request[controlv1.GetPostRepliesRequest]) (*connect.Response[controlv1.GetPostRepliesResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if req.Msg.GetPostId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("post_id is required"))
	}

	post, err := s.DB.GetPost(req.Msg.GetPostId())
	if err != nil || post == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("post not found"))
	}
	if !s.userCanReadSocialAccount(au, post.SocialAccountIDUint()) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read access to this post's social account is required"))
	}

	res := &controlv1.GetPostRepliesResponse{
		TrackingEnabled: s.postEngagementMaxAge() > 0,
	}

	if sa, saErr := s.DB.GetSocialAccount(post.SocialAccountIDUint()); saErr == nil && sa != nil {
		res.Supported = s.connectorSupportsReplies(sa.Connector)
	}

	stored, err := s.DB.GetPostEngagement(post.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load engagement"))
	}
	if stored != nil && stored.RepliesFetchedAt.Valid {
		res.LastFetchedAt = stored.RepliesFetchedAt.Time.UTC().Format(time.RFC3339)
	}

	rows, err := s.DB.SelectPostReplies(post.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load replies"))
	}

	res.ReplyCount = uint32(len(rows))
	res.Replies = replyNodesToProto(postreplies.BuildThread(rows))

	return connect.NewResponse(res), nil
}
