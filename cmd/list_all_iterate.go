package cmd

import (
	"context"
	"fmt"
	"os"

	"crmservice/internal/api"
)

type listAllPage struct {
	Data     []interface{}
	Included interface{}
	Page     int
	PageSize int
}

// listAllPageHandler processes one fetched list page. Returning an error stops iteration.
type listAllPageHandler func(page listAllPage) error

type listAllIterateResult struct {
	Total     int
	Truncated bool
}

func iterateListAllPages(
	ctx context.Context,
	client *api.Client,
	module string,
	baseOpts *api.ListOptions,
	pageSize int,
	maxResults int,
	verbose int,
	handler listAllPageHandler,
) (listAllIterateResult, error) {
	result := listAllIterateResult{}
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
			return result, err
		}

		pageData := toInterfaceSlice(resp.Data)
		if len(pageData) == 0 {
			break
		}

		truncated := false
		if maxResults > 0 {
			remaining := maxResults - result.Total
			if remaining <= 0 {
				result.Truncated = true
				break
			}
			if len(pageData) > remaining {
				pageData = pageData[:remaining]
				truncated = true
			}
		}

		if err := handler(listAllPage{
			Data:     pageData,
			Included: resp.Included,
			Page:     page,
			PageSize: pageSize,
		}); err != nil {
			return result, err
		}
		result.Total += len(pageData)

		if truncated {
			result.Truncated = true
			break
		}
		if maxResults > 0 && result.Total >= maxResults {
			break
		}
		if len(pageData) < pageSize {
			break
		}

		page++
	}

	if maxResults > 0 && result.Total == maxResults && !result.Truncated {
		hasMore, err := hasMoreListRecords(ctx, client, module, baseOpts, maxResults, verbose)
		if err != nil {
			return result, err
		}
		result.Truncated = hasMore
	}

	return result, nil
}