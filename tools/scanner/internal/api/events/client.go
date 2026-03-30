package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/infracost/cli/pkg/logging"
)

var (
	_ Client = (*client)(nil)
)

type Client interface {
	Push(ctx context.Context, event string, extra ...interface{})
}

type client struct {
	client *http.Client
	config *Config
}

func (c *client) Push(ctx context.Context, event string, extra ...interface{}) {
	if isTest, ok := metadata["isTest"].(bool); ok && isTest {
		return
	}

	if len(extra)%2 != 0 {
		panic("events.Push: extra args must be key-value pairs")
	}

	env := make(map[string]interface{}, len(metadata)+len(extra)/2)
	for k, v := range metadata {
		env[k] = v
	}
	for i := 0; i < len(extra); i += 2 {
		key, ok := extra[i].(string)
		if !ok {
			panic(fmt.Sprintf("events.Push: extra arg %d must be a string key", i))
		}
		env[key] = extra[i+1]
	}

	body := struct {
		Event string                 `json:"event"`
		Env   map[string]interface{} `json:"env"`
	}{
		Event: event,
		Env:   env,
	}

	buf, err := json.Marshal(body)
	if err != nil {
		logging.WithError(err).Msg("events: failed to marshal event")
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/event", c.config.Endpoint), bytes.NewReader(buf))
	if err != nil {
		logging.WithError(err).Msg("events: failed to create request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req) //nolint:gosec // endpoint is from CLI config, not user input
	if err != nil {
		logging.WithError(err).Msg("events: failed to send event")
		return
	}
	if err := resp.Body.Close(); err != nil {
		logging.WithError(err).Msg("events: failed to close response body")
		return
	}
}
