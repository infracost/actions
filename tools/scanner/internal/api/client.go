package api

import (
	"context"
	"net/http"

	"github.com/infracost/actions/tools/scanner/internal/trace"
	"golang.org/x/oauth2"
)

func Client(ctx context.Context, source oauth2.TokenSource, orgID string) *http.Client {
	base, transport := baseClient(ctx, source)
	return &http.Client{
		Transport: &CustomHeaders{
			Base: transport,
			Headers: map[string]string{
				"X-Infracost-Trace-ID": trace.ID,
				"User-Agent":           trace.UserAgent,
				"x-infracost-org-id":   orgID,
			},
		},
		CheckRedirect: base.CheckRedirect,
		Jar:           base.Jar,
		Timeout:       base.Timeout,
	}
}

func baseClient(ctx context.Context, source oauth2.TokenSource) (*http.Client, http.RoundTripper) {
	if source == nil {
		return http.DefaultClient, http.DefaultTransport
	}
	client := oauth2.NewClient(ctx, source)
	return client, client.Transport
}
