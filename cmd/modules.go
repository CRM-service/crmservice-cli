package cmd

import (
	"net/http"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func modulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modules",
		Short: "List API modules",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}
			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}
			full, err := cmd.Flags().GetBool("full")
			if err != nil {
				return err
			}
			verbose, err := cmd.Flags().GetInt("verbose")
			if err != nil {
				return err
			}
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}

			client := newAPIClient(url, token, verbose)

			var raw api.DiscoveryResponse
			if err := client.Do(cmd.Context(), http.MethodGet, "/", nil, &raw); err != nil {
				return output.ErrorResponse(err)
			}

			data := modulesDataFromLinks(raw.Links, full)

			respObj := &api.Response{
				Data:  data,
				Meta:  nil,
				Links: nil,
			}

			return output.ListResponse(respObj, output.Options{
				Format: outputFormat,
				Full:   full,
			})
		},
	}

	addCommonFlags(cmd, CommonFlagSet{Output: true, Verbose: true, Full: true})

	return cmd
}
