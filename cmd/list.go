package cmd

import (
	"crmservice/internal/api"
	filterpkg "crmservice/internal/filter"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <module>",
		Short: "List records",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListCommand(cmd, args, "")
		},
	}

	addListSearchFlags(cmd)
	cmd.Flags().String("filter", "", "Filter in JSON format")

	return cmd
}

func searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <module> <filter>",
		Short: "Search records (alias for list --filter)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := args[1]
			return runListCommand(cmd, args[:1], filter)
		},
	}

	addListSearchFlags(cmd)

	return cmd
}

func runListCommand(cmd *cobra.Command, args []string, filterOverride string) error {
	module := args[0]
	token, err := getRequiredTokenFromFlagEnvConfig(cmd)
	if err != nil {
		return err
	}
	common, err := readCommonFlags(cmd)
	if err != nil {
		return err
	}

	all, maxResults, err := validateListAllFlags(cmd)
	if err != nil {
		return err
	}

	pageSize, err := pageSizeForList(cmd, all)
	if err != nil {
		return err
	}
	page, err := cmd.Flags().GetInt("page")
	if err != nil {
		return err
	}
	offset, err := cmd.Flags().GetInt("offset")
	if err != nil {
		return err
	}
	if err := validateListPaginationFlags(cmd, page, offset, pageSize); err != nil {
		return err
	}
	fields, err := cmd.Flags().GetString("fields")
	if err != nil {
		return err
	}
	sort, err := cmd.Flags().GetString("sort")
	if err != nil {
		return err
	}
	filter := filterOverride
	if filter == "" {
		filter, err = cmd.Flags().GetString("filter")
		if err != nil {
			return err
		}
	}
	include, err := cmd.Flags().GetString("include")
	if err != nil {
		return err
	}

	url, err := getURLFromFlagOrEnv(cmd)
	if err != nil {
		return err
	}

	if filter != "" {
		if err := validateModuleFilter(module, filter, url, token, common.Verbose); err != nil {
			return output.ErrorResponse(err)
		}
	}

	apiClient := newAPIClient(url, token, common.Verbose)

	opts := &api.ListOptions{
		PageSize: pageSize,
	}

	if page > 0 {
		opts.SetPage(page)
	}

	if offset > 0 {
		opts.SetOffset(offset)
	}

	if fields != "" {
		opts.SetFields(splitCommaSeparated(fields))
	}

	if sort != "" {
		opts.SetSort(splitCommaSeparated(sort))
	}

	if filter != "" {
		filterObj, err := filterpkg.ParseToMap(filter)
		if err != nil {
			return err
		}
		opts.SetFilterObj(filterObj)
	}

	if include != "" {
		for _, rel := range splitCommaSeparated(include) {
			opts.AddInclude(rel)
		}
	}

	var outputFields []string
	if fields != "" {
		outputFields = splitCommaSeparated(fields)
	}

	if all {
		return runListAll(cmd, apiClient, module, opts, pageSize, maxResults, common.Verbose, common.Full, common.OutputFormat, outputFields)
	}

	resp, err := apiClient.List(cmd.Context(), module, opts)
	if err != nil {
		return output.ErrorResponse(err)
	}

	return output.ListResponse(resp, output.Options{
		Format: common.OutputFormat,
		Fields: outputFields,
		Full:   common.Full,
	})
}

func getCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <module> <id>",
		Short: "Get record by ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			module := args[0]
			id := args[1]
			token, err := getRequiredTokenFromFlagEnvConfig(cmd)
			if err != nil {
				return err
			}
			common, err := readCommonFlags(cmd)
			if err != nil {
				return err
			}

			url, err := getURLFromFlagOrEnv(cmd)
			if err != nil {
				return err
			}

			apiClient := newAPIClient(url, token, common.Verbose)

			opts := &api.ListOptions{}

			if fields, err := cmd.Flags().GetString("fields"); err == nil && fields != "" {
				opts.SetFields(splitCommaSeparated(fields))
			}

			resp, err := apiClient.Get(cmd.Context(), module, id, opts)
			if err != nil {
				return output.ErrorResponse(err)
			}

			var outputFields []string
			if fields, err := cmd.Flags().GetString("fields"); err == nil && fields != "" {
				outputFields = splitCommaSeparated(fields)
			}

			return output.ItemResponse(resp, output.Options{
				Format: common.OutputFormat,
				Fields: outputFields,
				Full:   common.Full,
			})
		},
	}

	cmd.Flags().String("fields", "", "Comma-separated field names to include (from 'attributes' branch)")
	addCommonFlags(cmd, CommonFlagSet{Output: true, Verbose: true, Full: true})

	return cmd
}