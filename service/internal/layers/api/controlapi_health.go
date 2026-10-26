package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"connectrpc.com/connect"
	armhealth "github.com/jamesread/armature-health"
	"github.com/jamesread/armature-health/store"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/health"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"github.com/jamesread/japella/internal/webhook"
)

const healthCheckInterval = 15 * time.Minute

func (s *ControlApi) initHealthService() {
	logLevel := ""
	if cfg := runtimeconfig.Get(); cfg != nil {
		logLevel = cfg.LogLevel
	}
	deps := health.Deps{
		DB:                       s.DB,
		Connectors:               s.cc,
		ErrorLog:                 utils.ErrorLog(),
		AdminUsesDefaultPassword: s.adminUsesDefaultPassword,
		Getenv:                   os.Getenv,
		LogLevel:                 logLevel,
	}
	s.health = armhealth.NewService(
		health.NewDBStore(s.DB, "mysql"),
		health.DefaultChecks(deps),
		health.WebhookNotifier{Dispatcher: webhook.NewDispatcher(s.DB)},
		armhealth.Options{},
	)
}

func (s *ControlApi) startHealthChecksLoop(recordJobRun func(string)) {
	go func() {
		timer := time.NewTimer(5 * time.Second)
		<-timer.C
		recordJobRun("health_checks")
		if s.health != nil {
			s.health.RunAll(context.Background())
		}

		ticker := time.NewTicker(healthCheckInterval)
		defer ticker.Stop()
		for range ticker.C {
			recordJobRun("health_checks")
			if s.health != nil {
				s.health.RunAll(context.Background())
			}
		}
	}()
}

func (s *ControlApi) adminUsesDefaultPassword(_ context.Context) (bool, error) {
	if s.DB == nil {
		return false, nil
	}
	adminUser := s.DB.GetUserByUsername("admin")
	if adminUser == nil || adminUser.PasswordHash == "" {
		return false, nil
	}
	return utils.VerifyPassword(adminUser.PasswordHash, "admin")
}

func (s *ControlApi) ListHealthIssues(ctx context.Context, req *connect.Request[controlv1.ListHealthIssuesRequest]) (*connect.Response[controlv1.ListHealthIssuesResponse], error) {
	if s.health == nil {
		return connect.NewResponse(&controlv1.ListHealthIssuesResponse{}), nil
	}
	issues, err := s.health.ListIssues(ctx, req.Msg.IncludeResolved, req.Msg.IncludeAcknowledged)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list health issues: %w", err))
	}
	checks, err := s.health.Statuses(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list health check runs: %w", err))
	}
	return connect.NewResponse(&controlv1.ListHealthIssuesResponse{
		Issues: toProtoHealthIssues(issues),
		Checks: toProtoHealthChecks(checks),
	}), nil
}

func (s *ControlApi) GetHealthSummary(ctx context.Context, _ *connect.Request[controlv1.GetHealthSummaryRequest]) (*connect.Response[controlv1.GetHealthSummaryResponse], error) {
	if s.health == nil {
		return connect.NewResponse(&controlv1.GetHealthSummaryResponse{}), nil
	}
	summary, err := s.health.Summary(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("health summary: %w", err))
	}
	return connect.NewResponse(&controlv1.GetHealthSummaryResponse{
		Errors:   clampInt32(summary.Errors),
		Warnings: clampInt32(summary.Warnings),
		Infos:    clampInt32(summary.Infos),
	}), nil
}

func clampInt32(n int) int32 {
	if n <= 0 {
		return 0
	}
	const maxInt32 = int(^uint32(0) >> 1)
	if n > maxInt32 {
		return int32(maxInt32)
	}
	return int32(n)
}

func (s *ControlApi) RunHealthChecks(ctx context.Context, req *connect.Request[controlv1.RunHealthChecksRequest]) (*connect.Response[controlv1.RunHealthChecksResponse], error) {
	if s.health == nil {
		return connect.NewResponse(&controlv1.RunHealthChecksResponse{}), nil
	}
	if err := s.runRequestedHealthChecks(ctx, req.Msg.CheckId); err != nil {
		return nil, err
	}
	statuses, err := s.health.Statuses(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list health check runs: %w", err))
	}
	return connect.NewResponse(&controlv1.RunHealthChecksResponse{
		Checks: toProtoHealthChecks(statuses),
	}), nil
}

func (s *ControlApi) runRequestedHealthChecks(ctx context.Context, checkID string) error {
	if checkID != "" {
		if err := s.health.Run(ctx, checkID); err != nil {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return nil
	}
	s.recordHealthChecksJobRun()
	s.health.RunAll(ctx)
	return nil
}

func (s *ControlApi) recordHealthChecksJobRun() {
	s.jobRunMutex.Lock()
	defer s.jobRunMutex.Unlock()
	if s.lastJobRunTimes != nil {
		s.lastJobRunTimes["health_checks"] = time.Now()
	}
}

func (s *ControlApi) AcknowledgeHealthIssue(ctx context.Context, req *connect.Request[controlv1.AcknowledgeHealthIssueRequest]) (*connect.Response[controlv1.AcknowledgeHealthIssueResponse], error) {
	if s.health == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("health checks are not enabled"))
	}
	if req.Msg.Id <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("issue id is required"))
	}
	err := s.health.Acknowledge(ctx, req.Msg.Id, req.Msg.Acknowledged, s.authenticatedUserID(ctx))
	if errors.Is(err, store.ErrIssueNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("acknowledge health issue: %w", err))
	}
	return connect.NewResponse(&controlv1.AcknowledgeHealthIssueResponse{}), nil
}

func (s *ControlApi) authenticatedUserID(ctx context.Context) int64 {
	if au := s.getAuthenticatedUser(ctx); au != nil && au.User != nil {
		return int64(au.User.ID)
	}
	return 0
}

func toProtoHealthIssues(issues []armhealth.IssueView) []*controlv1.HealthIssue {
	out := make([]*controlv1.HealthIssue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, &controlv1.HealthIssue{
			Id:                 issue.ID,
			CheckId:            issue.CheckID,
			CheckTitle:         issue.CheckTitle,
			IssueKey:           issue.IssueKey,
			Severity:           issue.Severity,
			Title:              issue.Title,
			Detail:             issue.Detail,
			Link:               issue.Link,
			FirstSeenUnix:      issue.FirstSeen,
			LastSeenUnix:       issue.LastSeen,
			Occurrences:        clampInt32(issue.Occurrences),
			ResolvedAtUnix:     issue.ResolvedAt.Int64,
			AcknowledgedAtUnix: issue.AcknowledgedAt.Int64,
		})
	}
	return out
}

func toProtoHealthChecks(checks []armhealth.CheckStatus) []*controlv1.HealthCheckStatus {
	out := make([]*controlv1.HealthCheckStatus, 0, len(checks))
	for _, check := range checks {
		out = append(out, &controlv1.HealthCheckStatus{
			CheckId:     check.CheckID,
			Title:       check.Title,
			LastRunUnix: check.LastRunAt,
			DurationMs:  check.DurationMs,
			Status:      check.Status,
			Error:       check.Error,
		})
	}
	return out
}
