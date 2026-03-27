package events

import (
	"net/http"
)

var (
	_ Config
)

type Config struct {
	Endpoint string `env:"INFRACOST_CLI_EVENTS_ENDPOINT" flag:"events-endpoint;hidden" usage:"The endpoint for the Infracost events service" default:"https://pricing.api.infracost.io"`

	ClientFn func(httpClient *http.Client) Client
}

func (c *Config) Client(httpClient *http.Client) Client {
	if c.ClientFn == nil {
		// The events client is kind of special in that it can be called anywhere in the CLI process (including outside
		// the context of commands). This means that it may not have been initialized, so we have this lazy evaluation /
		// initialization.
		c.ClientFn = func(httpClient *http.Client) Client {
			if c.Endpoint == "" {
				// if we're here outside the context of commands, then this may also have never been set.
				c.Endpoint = "https://pricing.api.infracost.io"
			}
			return &client{
				client: httpClient,
				config: c,
			}
		}
	}
	return c.ClientFn(httpClient)
}
