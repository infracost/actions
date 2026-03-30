package api

import "net/http"

var _ http.RoundTripper = (*CustomHeaders)(nil)

type CustomHeaders struct {
	Base    http.RoundTripper
	Headers map[string]string
}

func (h *CustomHeaders) RoundTrip(request *http.Request) (*http.Response, error) {
	for k, v := range h.Headers {
		if len(v) == 0 {
			continue // skip empty headers
		}
		request.Header.Set(k, v)
	}
	return h.Base.RoundTrip(request)
}
