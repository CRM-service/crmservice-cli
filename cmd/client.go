package cmd

import "crmservice/internal/api"

func newAPIClient(url, token string, verbose int) *api.Client {
	client := api.NewClient(url, token)
	client.Verbose = verbose
	client.HTTPClient.Timeout = getTimeoutFromConfig()
	return client
}
