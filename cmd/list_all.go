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
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.Flags().Int("page-size", 20, "Items per page (default 100 with --all)")
	cmd.Flags().String("include", "", "Comma-separated relation names to include")
	cmd.Flags().String("fields", "", "Comma-separated field names to include")
	cmd.Flags().String("sort", "", "Comma-separated field names to sort by (prefix with - for descending)")
	addCommonFlags(cmd, CommonFlagSet{Output: true, Verbose: true, Full: true})
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

func validateListPaginationFlags(cmd *cobra.Command, page, offset, pageSize int) error {
	if cmd.Flags().Changed("page") && page < 1 {
		return fmt.Errorf("--page must be at least 1")
	}
	if cmd.Flags().Changed("offset") && offset < 0 {
		return fmt.Errorf("--offset must be 0 or greater")
	}
	if pageSize < 1 {
		return fmt.Errorf("--page-size must be at least 1")
	}
	return nil
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
	iterateResult, err := iterateListAllPages(ctx, client, module, baseOpts, pageSize, maxResults, verbose, func(page listAllPage) error {
		result.data = append(result.data, page.Data...)
		if full && page.Included != nil {
			result.included = appendSlices(result.included, page.Included)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result.truncated = iterateResult.Truncated
	return result, nil
}

func hasMoreListRecords(ctx context.Context, client *api.Client, module string, baseOpts *api.ListOptions, offset int, verbose int) (bool, error) {
	if verbose >= 1 {
		fmt.Fprintf(os.Stderr, "[PAGE] Checking for records beyond max-results %d\n", offset)
	}

	opts := cloneListOptions(baseOpts)
	opts.Page = 0
	opts.PageSize = 1
	opts.SetOffset(offset)

	resp, err := client.List(ctx, module, opts)
	if err != nil {
		return false, err
	}
	return len(toInterfaceSlice(resp.Data)) > 0, nil
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

func dedupeIncludedResources(items []interface{}) []interface{} {
	seen := make(map[string]bool, len(items))
	deduped := make([]interface{}, 0, len(items))
	for _, item := range items {
		key := includedResourceKey(item)
		if key == "" {
			deduped = append(deduped, item)
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, item)
	}
	return deduped
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
	switch outputFormat {
	case "jsonl":
		return runListAllStreamJSONL(cmd, apiClient, module, baseOpts, pageSize, maxResults, verbose, full, outputFormat, outputFields)
	case "csv":
		return runListAllStreamCSV(cmd, apiClient, module, baseOpts, pageSize, maxResults, verbose, full, outputFormat, outputFields)
	}

	result, err := fetchAllListPages(cmd.Context(), apiClient, module, baseOpts, pageSize, maxResults, verbose, full)
	if err != nil {
		return output.ErrorResponse(err)
	}

	resp := &api.Response{Data: result.data}
	if full && len(result.included) > 0 {
		resp.Included = dedupeIncludedResources(result.included)
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

func runListAllStreamJSONL(
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
	streamOpts := output.Options{Format: "jsonl", Fields: outputFields, Full: full}
	seenIncluded := make(map[string]bool)

	iterateResult, err := iterateListAllPages(cmd.Context(), apiClient, module, baseOpts, pageSize, maxResults, verbose, func(page listAllPage) error {
		if err := output.StreamJSONLRecords(page.Data, streamOpts); err != nil {
			return err
		}
		if full && page.Included != nil {
			return streamNewIncludedResources(page.Included, seenIncluded, streamOpts)
		}
		return nil
	})
	if err != nil {
		return output.ErrorResponse(err)
	}

	if iterateResult.Total == 0 {
		fmt.Println("No data found")
	}
	if iterateResult.Truncated {
		return output.TruncationStatus(outputFormat, iterateResult.Total, maxResults)
	}
	return nil
}

func runListAllStreamCSV(
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
	streamOpts := output.Options{Format: "csv", Fields: outputFields, Full: full}
	writer := output.NewCSVStreamWriter()
	seenIncluded := make(map[string]bool)

	iterateResult, err := iterateListAllPages(cmd.Context(), apiClient, module, baseOpts, pageSize, maxResults, verbose, func(page listAllPage) error {
		if err := writer.WritePage(page.Data, streamOpts); err != nil {
			return err
		}
		if full && page.Included != nil {
			return streamNewIncludedResourcesCSV(page.Included, seenIncluded, writer, streamOpts)
		}
		return nil
	})
	if err != nil {
		return output.ErrorResponse(err)
	}

	if iterateResult.Total == 0 {
		fmt.Println("No data found")
	}
	if iterateResult.Truncated {
		return output.TruncationStatus(outputFormat, iterateResult.Total, maxResults)
	}
	return nil
}

func collectNewIncludedResources(included interface{}, seen map[string]bool) []interface{} {
	items := toInterfaceSlice(included)
	if len(items) == 0 {
		return nil
	}

	newItems := make([]interface{}, 0, len(items))
	for _, item := range items {
		key := includedResourceKey(item)
		if key == "" {
			newItems = append(newItems, item)
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		newItems = append(newItems, item)
	}
	return newItems
}

func streamNewIncludedResources(included interface{}, seen map[string]bool, opts output.Options) error {
	return output.StreamJSONLRecords(collectNewIncludedResources(included, seen), opts)
}

func streamNewIncludedResourcesCSV(included interface{}, seen map[string]bool, writer *output.CSVStreamWriter, opts output.Options) error {
	newItems := collectNewIncludedResources(included, seen)
	if len(newItems) == 0 {
		return nil
	}
	return writer.WritePage(newItems, opts)
}

func includedResourceKey(item interface{}) string {
	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return ""
	}
	id, idOK := itemMap["id"]
	resourceType, typeOK := itemMap["type"]
	if !idOK || !typeOK {
		return ""
	}
	return fmt.Sprintf("%v:%v", resourceType, id)
}
