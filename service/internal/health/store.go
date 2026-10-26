package health

import (
	"context"
	"errors"

	armhealth "github.com/jamesread/armature-health"
	"github.com/jamesread/armature-health/store"
	"github.com/jamesread/armature-health/store/sqlstore"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/webhook"
)

var errDBNotConnected = errors.New("database connection is not established")

// DBStore adapts Japella's reconnecting *db.DB to armature-health's store.Store.
type DBStore struct {
	DB     *db.DB
	Driver string
}

var _ store.Store = (*DBStore)(nil)

// NewDBStore returns a store.Store backed by database's active *sql.DB.
func NewDBStore(database *db.DB, driver string) *DBStore {
	return &DBStore{DB: database, Driver: driver}
}

func (s *DBStore) inner() (*sqlstore.SQL, error) {
	if s == nil || s.DB == nil {
		return nil, errDBNotConnected
	}
	raw := s.DB.SQLDB()
	if raw == nil {
		return nil, errDBNotConnected
	}
	return sqlstore.New(raw, s.Driver), nil
}

func (s *DBStore) ListOpenIssuesForCheck(ctx context.Context, checkID string) ([]store.Issue, error) {
	st, err := s.inner()
	if err != nil {
		return nil, err
	}
	return st.ListOpenIssuesForCheck(ctx, checkID)
}

func (s *DBStore) InsertIssue(ctx context.Context, issue store.Issue) (int64, error) {
	st, err := s.inner()
	if err != nil {
		return 0, err
	}
	return st.InsertIssue(ctx, issue)
}

func (s *DBStore) UpdateSeenIssue(ctx context.Context, issue store.Issue) error {
	st, err := s.inner()
	if err != nil {
		return err
	}
	return st.UpdateSeenIssue(ctx, issue)
}

func (s *DBStore) ResolveIssue(ctx context.Context, id int64, at int64) error {
	st, err := s.inner()
	if err != nil {
		return err
	}
	return st.ResolveIssue(ctx, id, at)
}

func (s *DBStore) ListIssues(ctx context.Context, includeResolved, includeAcknowledged bool) ([]store.Issue, error) {
	st, err := s.inner()
	if err != nil {
		return nil, err
	}
	return st.ListIssues(ctx, includeResolved, includeAcknowledged)
}

func (s *DBStore) SetIssueAcknowledged(ctx context.Context, id int64, acknowledged bool, userID int64, at int64) error {
	st, err := s.inner()
	if err != nil {
		return err
	}
	return st.SetIssueAcknowledged(ctx, id, acknowledged, userID, at)
}

func (s *DBStore) Summary(ctx context.Context) (store.Summary, error) {
	st, err := s.inner()
	if err != nil {
		return store.Summary{}, err
	}
	return st.Summary(ctx)
}

func (s *DBStore) PruneResolvedIssues(ctx context.Context, before int64) error {
	st, err := s.inner()
	if err != nil {
		return err
	}
	return st.PruneResolvedIssues(ctx, before)
}

func (s *DBStore) SaveCheckRun(ctx context.Context, run store.CheckRun) error {
	st, err := s.inner()
	if err != nil {
		return err
	}
	return st.SaveCheckRun(ctx, run)
}

func (s *DBStore) ListCheckRuns(ctx context.Context) ([]store.CheckRun, error) {
	st, err := s.inner()
	if err != nil {
		return nil, err
	}
	return st.ListCheckRuns(ctx)
}

// WebhookNotifier adapts webhook.Dispatcher to armature-health's Notifier interface.
type WebhookNotifier struct {
	Dispatcher *webhook.Dispatcher
}

var _ armhealth.Notifier = WebhookNotifier{}

func (n WebhookNotifier) SendNotification(ctx context.Context, eventCode string, data map[string]any) error {
	if n.Dispatcher != nil {
		n.Dispatcher.Dispatch(ctx, eventCode, data)
	}
	return nil
}
