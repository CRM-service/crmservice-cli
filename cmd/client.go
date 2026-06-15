package cmd

import (
	"time"

	"crmservice/internal/api"
)

func newAPIClient(url, token string, verbose int) *api.Client {
	return newAPIClientWithTimeout(url, token, verbose, getTimeoutFromConfig())
}

func newAPIClientWithTimeout(url, token string, verbose int, timeout time.Duration) *api.Client {
	client := api.NewClient(url, token)
	client.Verbose = verbose
	client.HTTPClient.Timeout = timeout
	return client
}
