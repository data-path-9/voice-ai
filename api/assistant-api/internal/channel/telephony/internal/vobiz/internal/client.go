// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vobiz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/rapidaai/pkg/clients/rest"
)

const (
	// DefaultBaseURL is the Vobiz API base URL.
	DefaultBaseURL = "https://api.vobiz.ai"
	// DefaultTimeoutSeconds is the default timeout for Vobiz REST API calls.
	DefaultTimeoutSeconds uint32 = 30

	makeCallPathFormat = "/api/v1/Account/%s/Call/"
)

// Client is the minimal Vobiz call surface used by the vobiz provider.
type Client interface {
	// MakeCall originates an outbound call via the Vobiz REST API.
	MakeCall(ctx context.Context, authID, authToken string, req MakeCallRequest) (*CallResponse, error)
}

type client struct {
	http rest.APIClient
}

type Options struct {
	BaseURL        string
	TimeoutSeconds uint32
	HTTPClient     rest.APIClient
}

type FuncOption func(*Options)

func WithBaseURL(baseURL string) FuncOption {
	return func(options *Options) {
		options.BaseURL = baseURL
	}
}

func WithTimeoutSeconds(timeoutSeconds uint32) FuncOption {
	return func(options *Options) {
		options.TimeoutSeconds = timeoutSeconds
	}
}

func WithAPIClient(httpClient rest.APIClient) FuncOption {
	return func(options *Options) {
		options.HTTPClient = httpClient
	}
}

// New returns a Vobiz client. By default it targets production with a 30s timeout.
func New(opts ...FuncOption) Client {
	options := Options{
		BaseURL:        DefaultBaseURL,
		TimeoutSeconds: DefaultTimeoutSeconds,
	}
	for _, opt := range opts {
		opt(&options)
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = rest.NewRestClientWithConfig(options.BaseURL, nil, options.TimeoutSeconds)
	}
	return &client{http: httpClient}
}

func (c *client) MakeCall(ctx context.Context, authID, authToken string, req MakeCallRequest) (*CallResponse, error) {
	// Credentials are tenant-supplied (from the integration form), not app config.
	resp, err := c.http.Post(ctx,
		fmt.Sprintf(makeCallPathFormat, url.PathEscape(authID)),
		req,
		map[string]string{
			"X-Auth-ID":    authID,
			"X-Auth-Token": authToken,
			"Content-Type": "application/json",
		})
	if err != nil {
		return nil, err
	}

	raw := resp.Body
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := ""
		var parsed map[string]interface{}
		if json.Unmarshal(raw, &parsed) == nil {
			for _, k := range []string{"message", "error", "error_message", "detail", "errors"} {
				if v, ok := parsed[k]; ok && v != nil {
					msg = fmt.Sprintf("%v", v)
					break
				}
			}
		}
		return nil, &VobizAPIError{StatusCode: resp.StatusCode, Body: string(raw), Message: msg}
	}

	var call CallResponse
	if err := json.Unmarshal(raw, &call); err != nil {
		return nil, fmt.Errorf("failed to parse vobiz call response: %w", err)
	}
	return &call, nil
}
