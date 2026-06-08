package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxResponseBodySize = 32 << 20 // 32 MiB

func readResponseBody(r io.Reader) ([]byte, error) {
	limited := io.LimitReader(r, maxResponseBodySize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxResponseBodySize {
		return nil, fmt.Errorf("response body exceeds limit of %d bytes", maxResponseBodySize)
	}
	return body, nil
}

type Client struct {
	BaseURL        string
	AuthToken      string
	HTTPClient     *http.Client
	DefaultHeaders map[string]string
	Verbose        int
}

type Error struct {
	Status  int
	Body    []byte
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Status, e.Message)
}

func NewClient(baseURL, authToken string) *Client {
	return &Client{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		AuthToken:  authToken,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		DefaultHeaders: map[string]string{
			"Content-Type": "application/vnd.api+json",
			"Accept":       "application/vnd.api+json",
		},
	}
}

var sensitiveLogHeaderNames = map[string]bool{
	"authorization": true,
	"cookie":        true,
	"set-cookie":    true,
}

func (c *Client) requestHeaders(method string) map[string]string {
	headers := make(map[string]string, len(c.DefaultHeaders))
	for key, value := range c.DefaultHeaders {
		if method == http.MethodGet && strings.EqualFold(key, "Content-Type") {
			continue
		}
		headers[key] = value
	}
	return headers
}

func (c *Client) outboundRequestHeaders(method string) map[string]string {
	headers := c.requestHeaders(method)
	if c.AuthToken != "" {
		headers["Authorization"] = "Bearer " + c.AuthToken
	}
	return headers
}

func redactHeaders(headers map[string]string) map[string]string {
	redacted := make(map[string]string, len(headers))
	for key, value := range headers {
		if sensitiveLogHeaderNames[strings.ToLower(key)] {
			redacted[key] = "[redacted]"
			continue
		}
		redacted[key] = value
	}
	return redacted
}

func (c *Client) logRequest(method, path string) {
	if c.Verbose >= 1 {
		reqURL := c.BaseURL + "/" + strings.TrimPrefix(path, "/")
		fmt.Fprintf(os.Stderr, "[REQUEST] %s %s\n", method, reqURL)
	}
	if c.Verbose >= 2 {
		fmt.Fprintf(os.Stderr, "[REQUEST HEADERS] %v\n", redactHeaders(c.outboundRequestHeaders(method)))
	}
}

func (c *Client) logResponse(status int, body []byte) {
	if c.Verbose >= 1 {
		fmt.Fprintf(os.Stderr, "[RESPONSE] Status: %d\n", status)
	}
	if c.Verbose >= 2 {
		fmt.Fprintf(os.Stderr, "[RESPONSE BODY]\n%s\n", string(body))
	}
}

func prepareRequestBody(body interface{}) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(jsonBody), nil
}

type ListOptions struct {
	Page     int
	PageSize int
	Offset   int
	Fields   []string
	Filter   map[string]interface{}
	Include  []string
	Sort     []string
}

func NewListOptions() *ListOptions {
	return &ListOptions{}
}

func (o *ListOptions) SetPageSize(size int) *ListOptions {
	o.PageSize = size
	return o
}

func (o *ListOptions) SetPage(page int) *ListOptions {
	o.Page = page
	return o
}

func (o *ListOptions) SetOffset(offset int) *ListOptions {
	o.Offset = offset
	return o
}

func (o *ListOptions) SetFields(fields []string) *ListOptions {
	o.Fields = fields
	return o
}

func (o *ListOptions) SetFilterObj(filterObj map[string]interface{}) {
	o.Filter = filterObj
}

func (o *ListOptions) AddFilter(field, value string) *ListOptions {
	if o.Filter == nil {
		o.Filter = make(map[string]interface{})
	}
	o.Filter[field] = value
	return o
}

func (o *ListOptions) AddInclude(relation string) *ListOptions {
	o.Include = append(o.Include, relation)
	return o
}

func (o *ListOptions) SetSort(fields []string) *ListOptions {
	o.Sort = fields
	return o
}

type Meta struct {
	Total    int `json:"total,omitempty"`
	Page     int `json:"page,omitempty"`
	PageSize int `json:"page_size,omitempty"`
}

type Links struct {
	Self  string `json:"self,omitempty"`
	First string `json:"first,omitempty"`
	Prev  string `json:"prev,omitempty"`
	Next  string `json:"next,omitempty"`
	Last  string `json:"last,omitempty"`
}

type Response struct {
	Data     interface{} `json:"data"`
	Meta     *Meta       `json:"meta,omitempty"`
	Links    *Links      `json:"links,omitempty"`
	Included interface{} `json:"included,omitempty"`
}

type SingleResponse struct {
	Data     interface{} `json:"data"`
	Meta     *Meta       `json:"meta,omitempty"`
	Links    *Links      `json:"links,omitempty"`
	Included interface{} `json:"included,omitempty"`
}

func (c *Client) Do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	c.logRequest(method, path)

	reqURL := c.BaseURL + "/" + strings.TrimPrefix(path, "/")
	reqBody, err := prepareRequestBody(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return err
	}

	for key, value := range c.requestHeaders(method) {
		req.Header.Set(key, value)
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := readResponseBody(resp.Body)
	if err != nil {
		return err
	}

	c.logResponse(resp.StatusCode, respBody)

	if resp.StatusCode >= 400 {
		apiErr := &Error{
			Status:  resp.StatusCode,
			Body:    respBody,
			Message: string(respBody),
		}
		return apiErr
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) List(ctx context.Context, module string, opts *ListOptions) (*Response, error) {
	path := "/" + module
	query := url.Values{}

	if opts != nil {
		if opts.Page > 0 {
			query.Set("page[number]", fmt.Sprintf("%d", opts.Page))
		}
		if opts.PageSize > 0 {
			query.Set("page[size]", fmt.Sprintf("%d", opts.PageSize))
		}
		if opts.Offset > 0 {
			query.Set("offset", fmt.Sprintf("%d", opts.Offset))
		}
		if len(opts.Fields) > 0 {
			query.Set("fields["+module+"]", strings.Join(opts.Fields, ","))
		}
		if opts.Filter != nil {
			filterJSON, err := json.Marshal(opts.Filter)
			if err != nil {
				return nil, err
			}
			query.Set("filter", string(filterJSON))
		}
		if len(opts.Include) > 0 {
			query.Set("include", strings.Join(opts.Include, ","))
		}
		if len(opts.Sort) > 0 {
			query.Set("sort", strings.Join(opts.Sort, ","))
		}
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var resp Response
	err := c.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Get(ctx context.Context, module, id string, opts *ListOptions) (*SingleResponse, error) {
	path := fmt.Sprintf("/%s/%s", module, id)

	query := url.Values{}
	if opts != nil && len(opts.Fields) > 0 {
		query.Set("fields["+module+"]", strings.Join(opts.Fields, ","))
	}
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var resp SingleResponse
	err := c.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Create(ctx context.Context, module string, data interface{}) (*SingleResponse, error) {
	path := "/" + module
	requestBody := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       module,
			"attributes": data,
		},
	}
	var resp SingleResponse
	err := c.Do(ctx, "POST", path, requestBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Update(ctx context.Context, module, id string, data interface{}) (*SingleResponse, error) {
	path := fmt.Sprintf("/%s/%s", module, id)
	requestBody := map[string]interface{}{
		"data": map[string]interface{}{
			"type":       module,
			"id":         id,
			"attributes": data,
		},
	}
	var resp SingleResponse
	err := c.Do(ctx, "PATCH", path, requestBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Delete(ctx context.Context, module, id string) error {
	path := fmt.Sprintf("/%s/%s", module, id)
	return c.Do(ctx, "DELETE", path, nil, nil)
}
