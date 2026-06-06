package cmd

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

const listAllDefaultPageSize = 100

func addListSearchFlags(cmd *cobra.Command) {
	cmd.Flags().Int("page-size", 20, "Items per page (default 100 with --all)")
	cmd.Flags().String("include", "", "Comma-separated relation names to include")
	cmd.Flags().String("fields", "", "Comma-separated field names to include")
	cmd.Flags().String("sort", "", "Comma-separated field names to sort by (prefix with - for descending)")
	cmd.Flags().StringP("output", "o", "table", "Output format: table, json, yaml, jsonl, or csv")
	cmd.Flags().Bool("full", false, "Include full response (not just attributes)")
	cmd.Flags().Int("verbose", 0, "Verbose output level (0=quiet, 1=REQUEST/RESPONSE summary, 2=detailed)")
	cmd.Flags().Int("page", 1, "Page number")
	cmd.Flags().Int("offset", 0, "Offset for pagination")
	cmd.Flags().Bool("all", false, "Fetch all pages automatically")
	cmd.Flags().Int("max-results", 0, "Maximum records to return with --all (0 = unlimited); requires --all")
}

func validateListAllFlags(cmd *cobra.Command) (bool, int, error) {
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return false, 0, err
	}

	maxResultsChanged := cmd.Flags().Changed("max-results")
	if maxResultsChanged && !all {
		return false, 0, fmt.Errorf("--max-results requires --all")
	}
	if all && !maxResultsChanged {
		return false, 0, fmt.Errorf("--all requires --max-results (use 0 for unlimited)")
	}

	if all {
		if cmd.Flags().Changed("page") {
			return false, 0, fmt.Errorf("--page cannot be used with --all")
		}
		if cmd.Flags().Changed("offset") {
			return false, 0, fmt.Errorf("--offset cannot be used with --all")
		}
	}

	maxResults := 0
	if maxResultsChanged {
		maxResults, err = cmd.Flags().GetInt("max-results")
		if err != nil {
			return false, 0, err
		}
		if maxResults < 0 {
			return false, 0, fmt.Errorf("--max-results must be 0 or greater")
		}
	}

	return all, maxResults, nil
}

func pageSizeForList(cmd *cobra.Command, all bool) (int, error) {
	if all && !cmd.Flags().Changed("page-size") {
		return listAllDefaultPageSize, nil
	}
	return getPageSizeFromFlagConfig(cmd)
}

type listAllResult struct {
	data      []interface{}
	included  []interface{}
	truncated bool
}

func fetchAllListPages(
	ctx context.Context,
	client *api.Client,
	module string,
	baseOpts *api.ListOptions,
	pageSize int,
	maxResults int,
	verbose int,
	full bool,
) (*listAllResult, error) {
	result := &listAllResult{}
	page := 1

	for {
		if verbose >= 1 {
			fmt.Fprintf(os.Stderr, "[PAGE] Fetching page %d (page-size %d)\n", page, pageSize)
		}

		opts := cloneListOptions(baseOpts)
		opts.PageSize = pageSize
		opts.SetPage(page)
		opts.Offset = 0

		resp, err := client.List(ctx, module, opts)
		if err != nil {
			return nil, err
		}

		pageData := toInterfaceSlice(resp.Data)
		if len(pageData) == 0 {
			break
		}

		if maxResults > 0 {
			remaining := maxResults - len(result.data)
			if remaining <= 0 {
				result.truncated = true
				break
			}
			if len(pageData) > remaining {
				pageData = pageData[:remaining]
				result.truncated = true
			}
		}

		result.data = append(result.data, pageData...)

		if full && resp.Included != nil {
			result.included = appendSlices(result.included, resp.Included)
		}

		if result.truncated {
			break
		}
		if maxResults > 0 && len(result.data) >= maxResults && len(pageData) == pageSize {
			result.truncated = true
			break
		}
		if len(pageData) < pageSize {
			break
		}

		page++
	}

	return result, nil
}

func cloneListOptions(opts *api.ListOptions) *api.ListOptions {
	if opts == nil {
		return api.NewListOptions()
	}
	clone := *opts
	if len(opts.Fields) > 0 {
		clone.Fields = append([]string(nil), opts.Fields...)
	}
	if len(opts.Include) > 0 {
		clone.Include = append([]string(nil), opts.Include...)
	}
	if len(opts.Sort) > 0 {
		clone.Sort = append([]string(nil), opts.Sort...)
	}
	if opts.Filter != nil {
		clone.Filter = opts.Filter
	}
	return &clone
}

func toInterfaceSlice(data interface{}) []interface{} {
	if data == nil {
		return nil
	}

	dataVal := reflect.ValueOf(data)
	if dataVal.Kind() != reflect.Slice {
		return nil
	}

	items := make([]interface{}, dataVal.Len())
	for i := 0; i < dataVal.Len(); i++ {
		items[i] = dataVal.Index(i).Interface()
	}
	return items
}

func appendSlices(dst []interface{}, src interface{}) []interface{} {
	return append(dst, toInterfaceSlice(src)...)
}

func runListAll(
	cmd *cobra.Command,
	apiClient *api.Client,
	module string,
	baseOpts *api.ListOptions,
	pageSize int,
	maxResults int,
	verbose int,
	full bool,
	outputFormat string,
	outputFields []string,
) error {
	result, err := fetchAllListPages(cmd.Context(), apiClient, module, baseOpts, pageSize, maxResults, verbose, full)
	if err != nil {
		return output.ErrorResponse(err)
	}

	resp := &api.Response{Data: result.data}
	if full && len(result.included) > 0 {
		resp.Included = result.included
	}

	if err := output.ListResponse(resp, output.Options{
		Format: outputFormat,
		Fields: outputFields,
		Full:   full,
	}); err != nil {
		return err
	}

	if result.truncated {
		return output.TruncationStatus(outputFormat, len(result.data), maxResults)
	}
	return nil
}
