package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	HeaderEvent     = "X-Japella-Event"
	HeaderSignature = "X-Japella-Signature"
)

type Target struct {
	ID     uint32
	URL    string
	Secret string
}

type TargetStore interface {
	EnabledTargetsForEvent(ctx context.Context, event string) ([]Target, error)
}

// DeliveryRecord is one outbound attempt to a webhook target.
type DeliveryRecord struct {
	WebhookTargetID uint32
	Event           string
	Success         bool
	HTTPStatus      int
	ErrorMessage    string
	DurationMs      int
}

// DeliveryRecorder persists delivery attempts for the webhook detail UI.
type DeliveryRecorder interface {
	RecordWebhookDelivery(ctx context.Context, entry DeliveryRecord) error
}

type Dispatcher struct {
	Store    TargetStore
	Recorder DeliveryRecorder
	Client   *http.Client
}

func NewDispatcher(store TargetStore) *Dispatcher {
	d := &Dispatcher{
		Store: store,
		Client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
	if recorder, ok := store.(DeliveryRecorder); ok {
		d.Recorder = recorder
	}
	return d
}

func (d *Dispatcher) Dispatch(ctx context.Context, event string, payload map[string]any) {
	if d == nil || d.Store == nil {
		return
	}
	event, targets, raw, ok := d.prepareDispatch(ctx, event, payload)
	if !ok {
		return
	}
	client := d.httpClient(2 * time.Second)
	for _, target := range targets {
		go d.deliver(context.Background(), client, target, event, raw) //nolint:gosec // deliveries must outlive the request that triggered them
	}
}

func (d *Dispatcher) prepareDispatch(ctx context.Context, event string, payload map[string]any) (string, []Target, []byte, bool) {
	event, err := NormalizeEvent(event)
	if err != nil {
		log.Warnf("webhook dispatch: %v", err)
		return "", nil, nil, false
	}
	targets, err := d.Store.EnabledTargetsForEvent(ctx, event)
	if err != nil {
		log.Warnf("webhook dispatch: list targets for %s: %v", event, err)
		return "", nil, nil, false
	}
	if len(targets) == 0 {
		return "", nil, nil, false
	}
	raw, err := d.marshalBody(event, payload)
	if err != nil {
		log.Warnf("webhook dispatch: marshal payload: %v", err)
		return "", nil, nil, false
	}
	return event, targets, raw, true
}

func (d *Dispatcher) httpClient(fallbackTimeout time.Duration) *http.Client {
	if d.Client != nil {
		return d.Client
	}
	return &http.Client{Timeout: fallbackTimeout}
}

// DeliverOne POSTs a signed event payload to a single target and returns the HTTP status.
func (d *Dispatcher) DeliverOne(ctx context.Context, target Target, event string, payload map[string]any) (int, error) {
	if d == nil {
		return 0, fmt.Errorf("webhook dispatcher is nil")
	}
	event, err := NormalizeEvent(event)
	if err != nil {
		return 0, err
	}
	raw, err := d.marshalBody(event, payload)
	if err != nil {
		return 0, err
	}
	return d.deliverSync(ctx, d.clientForContext(ctx, 10*time.Second), target, event, raw)
}

func (d *Dispatcher) clientForContext(ctx context.Context, fallbackTimeout time.Duration) *http.Client {
	client := d.httpClient(fallbackTimeout)
	if ctx == nil {
		return client
	}
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) <= 0 {
		return client
	}
	c := *client
	c.Timeout = time.Until(deadline)
	return &c
}

func (d *Dispatcher) marshalBody(event string, payload map[string]any) ([]byte, error) {
	body := map[string]any{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	maps.Copy(body, payload)
	return json.Marshal(body)
}

func (d *Dispatcher) deliver(ctx context.Context, client *http.Client, target Target, event string, raw []byte) {
	_, _ = d.deliverSync(ctx, client, target, event, raw)
}

func (d *Dispatcher) deliverSync(ctx context.Context, client *http.Client, target Target, event string, raw []byte) (int, error) {
	started := time.Now()
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := d.newSignedRequest(ctx, target, event, raw)
	if err != nil {
		d.record(ctx, target.ID, event, false, 0, err.Error(), started)
		log.Warnf("webhook target %d: create request: %v", target.ID, err)
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		d.record(ctx, target.ID, event, false, 0, err.Error(), started)
		log.Warnf("webhook target %d: POST %s: %v", target.ID, target.URL, err)
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	d.recordDeliveryStatus(ctx, target, event, resp.StatusCode, started)
	return resp.StatusCode, nil
}

func (d *Dispatcher) newSignedRequest(ctx context.Context, target Target, event string, raw []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.URL, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderEvent, event)
	req.Header.Set(HeaderSignature, fmt.Sprintf("sha256=%s", Signature(string(raw), target.Secret)))
	return req, nil
}

func (d *Dispatcher) recordDeliveryStatus(ctx context.Context, target Target, event string, status int, started time.Time) {
	ok := status >= 200 && status < 300
	errMsg := ""
	if !ok {
		errMsg = fmt.Sprintf("HTTP %d", status)
		log.Warnf("webhook target %d: POST %s returned %d", target.ID, target.URL, status)
	}
	d.record(ctx, target.ID, event, ok, status, errMsg, started)
}

func (d *Dispatcher) record(ctx context.Context, targetID uint32, event string, success bool, httpStatus int, errMsg string, started time.Time) {
	if d == nil || d.Recorder == nil {
		return
	}
	duration := max(int(time.Since(started).Milliseconds()), 0)
	if len(errMsg) > 1000 {
		errMsg = errMsg[:1000]
	}
	if err := d.Recorder.RecordWebhookDelivery(ctx, DeliveryRecord{
		WebhookTargetID: targetID,
		Event:           event,
		Success:         success,
		HTTPStatus:      httpStatus,
		ErrorMessage:    errMsg,
		DurationMs:      duration,
	}); err != nil {
		log.Warnf("webhook target %d: record delivery: %v", targetID, err)
	}
}
