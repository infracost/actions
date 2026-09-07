package dashboard

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavePostedPrComment(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		response   string
		wantSaved  bool
		wantErrMsg string
	}{
		{
			name:      "saved",
			status:    http.StatusOK,
			response:  `{"data":{"savePostedPrComment":true}}`,
			wantSaved: true,
		},
		{
			name:     "not saved",
			status:   http.StatusOK,
			response: `{"data":{"savePostedPrComment":false}}`,
		},
		{
			name:       "graphql errors",
			status:     http.StatusOK,
			response:   `{"errors":[{"message":"run not found"},{"message":"forbidden"}]}`,
			wantErrMsg: "run not found; forbidden",
		},
		{
			name:       "missing field",
			status:     http.StatusOK,
			response:   `{"data":{}}`,
			wantErrMsg: "savePostedPrComment missing from response",
		},
		{
			name:       "null data",
			status:     http.StatusOK,
			response:   `{"data":null}`,
			wantErrMsg: "savePostedPrComment missing from response",
		},
		{
			name:       "gateway error",
			status:     http.StatusBadGateway,
			response:   `{"message":"Internal server error"}`,
			wantErrMsg: "savePostedPrComment missing from response",
		},
		{
			name:       "forbidden",
			status:     http.StatusForbidden,
			response:   `{"message":"forbidden"}`,
			wantErrMsg: "savePostedPrComment missing from response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/graphql", r.URL.Path)

				body, err := io.ReadAll(r.Body)
				if !assert.NoError(t, err) {
					return
				}

				var request struct {
					Query     string            `json:"query"`
					Variables map[string]string `json:"variables"`
				}
				if !assert.NoError(t, json.Unmarshal(body, &request)) {
					return
				}

				assert.Contains(t, request.Query, "mutation SavePostedPrComment($runId: String!, $comment: String!)")
				assert.Contains(t, request.Query, "savePostedPrComment(runId: $runId, comment: $comment)")
				assert.Equal(t, map[string]string{"runId": "test-run-id", "comment": "comment body"}, request.Variables)

				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			c := newTestClient(server.URL)

			saved, err := c.SavePostedPrComment(context.Background(), "test-run-id", "comment body")

			if tt.wantErrMsg != "" {
				require.EqualError(t, err, tt.wantErrMsg)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantSaved, saved)
		})
	}
}

func TestSavePostedPrCommentValidation(t *testing.T) {
	tests := []struct {
		name       string
		runID      string
		comment    string
		wantErrMsg string
	}{
		{
			name:       "empty run id",
			comment:    "comment body",
			wantErrMsg: "runID is required",
		},
		{
			name:       "empty comment",
			runID:      "test-run-id",
			wantErrMsg: "comment is required",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		assert.Fail(t, "unexpected request to the dashboard")
	}))
	defer server.Close()

	c := newTestClient(server.URL)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved, err := c.SavePostedPrComment(context.Background(), tt.runID, tt.comment)
			require.EqualError(t, err, tt.wantErrMsg)
			assert.False(t, saved)
		})
	}
}

func TestSavePostedPrCommentTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if !assert.NoError(t, err) {
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()

	c := newTestClient(server.URL)

	saved, err := c.SavePostedPrComment(context.Background(), "test-run-id", "comment body")
	require.Error(t, err)
	assert.False(t, saved)
}

func newTestClient(endpoint string) Client {
	cfg := &Config{Endpoint: endpoint}
	cfg.Process()
	return cfg.Client(http.DefaultClient)
}
