package internal_custom_llm_common

import (
	"net/http"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func NewHTTPClient(config ClientConfig) *http.Client {
	transport := &http.Transport{
		ForceAttemptHTTP2:   true,
		MaxConnsPerHost:     StreamMaxConnsPerHost,
		MaxIdleConnsPerHost: StreamMaxIdleConnsPerHost,
		MaxIdleConns:        StreamMaxIdleConns,
		IdleConnTimeout:     StreamIdleConnTimeout,
	}
	return &http.Client{
		Timeout:   config.ClientTimeout,
		Transport: transport,
	}
}

func NewOpenAIClientWithHTTPClient(config ClientConfig) (openai.Client, *http.Client) {
	httpClient := NewHTTPClient(config)
	opts := []option.RequestOption{
		option.WithHTTPClient(httpClient),
		option.WithRequestTimeout(config.RequestTimeout),
		option.WithBaseURL(config.BaseURL),
	}
	for key, value := range config.Headers {
		opts = append(opts, option.WithHeader(key, value))
	}
	return openai.NewClient(opts...), httpClient
}

func NewOpenAIClient(config ClientConfig) openai.Client {
	client, _ := NewOpenAIClientWithHTTPClient(config)
	return client
}

func ApplyHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		if key == "" {
			continue
		}
		req.Header.Set(key, value)
	}
}
