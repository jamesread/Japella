package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

type Dispatcher struct {
	Store  TargetStore
	Client *http.Client
}

func NewDispatcher(store TargetStore) *Dispatcher {
	return &Dispatcher{
		Store: store,
		Client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, event string, payload map[string]any) {
	if d == nil || d.Store == nil {
		return
	}
	event, err := NormalizeEvent(event)
	if err != nil {
		log.Warnf("webhook dispatch: %v", err)
		return
	}
	targets, err := d.Store.EnabledTargetsForEvent(ctx, event)
	if err != nil {
		log.Warnf("webhook dispatch: list targets for %s: %v", event, err)
		return
	}
	if len(targets) == 0 {
		return
	}

	body := map[string]any{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range payload {
		body[k] = v
	}

	raw, err := json.Marshal(body)
	if err != nil {
		log.Warnf("webhook dispatch: marshal payload: %v", err)
		return
	}

	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}

	for _, target := range targets {
		go d.deliver(client, target, event, raw)
	}
}

func (d *Dispatcher) deliver(client *http.Client, target Target, event string, raw []byte) {
	req, err := http.NewRequest(http.MethodPost, target.URL, bytes.NewReader(raw))
	if err != nil {
		log.Warnf("webhook target %d: create request: %v", target.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderEvent, event)
	req.Header.Set(HeaderSignature, fmt.Sprintf("sha256=%s", Signature(string(raw), target.Secret)))

	resp, err := client.Do(req)
	if err != nil {
		log.Warnf("webhook target %d: POST %s: %v", target.ID, target.URL, err)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warnf("webhook target %d: POST %s returned %d", target.ID, target.URL, resp.StatusCode)
	}
}
