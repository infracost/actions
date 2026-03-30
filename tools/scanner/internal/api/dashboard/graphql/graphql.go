package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type Response[T any] struct {
	Data   T       `json:"data"`
	Errors []Error `json:"errors,omitempty"`
}

type Error struct {
	Message string `json:"message"`
}

type Request struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

func Query[T any](ctx context.Context, client *http.Client, endpoint string, query string, variables map[string]interface{}) (Response[T], error) {
	request := Request{
		Query:     query,
		Variables: variables,
	}

	bytes := new(bytes.Buffer)
	if err := json.NewEncoder(bytes).Encode(request); err != nil {
		return Response[T]{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes)
	if err != nil {
		return Response[T]{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	r, err := client.Do(req) // #nosec G704 -- request target originates from config file
	if err != nil {
		return Response[T]{}, err
	}
	defer func() {
		_ = r.Body.Close()
	}()

	var response Response[T]
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return Response[T]{}, err
	}

	return response, nil
}
