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

// DiscoveryResponse is the JSON:API root document listing module links.
type DiscoveryResponse struct {
	Meta  interface{}            `json:"meta,omitempty"`
	Links map[string]interface{} `json:"links,omitempty"`
}

type JSONAPIResponse struct {
	Data     interface{} `json:"data"`
	Meta     *Meta       `json:"meta,omitempty"`
	Links    *Links      `json:"links,omitempty"`
	Included interface{} `json:"included,omitempty"`
}

// Response and SingleResponse are aliases for the shared JSON:API envelope type.
type Response = JSONAPIResponse
type SingleResponse = JSONAPIResponse

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
	attrs, ok := data.(map[string]interface{})
	if !ok {
		attrs = map[string]interface{}{}
	}
	requestBody := BuildSingleWriteBody(module, "", "create", attrs)
	var resp SingleResponse
	err := c.Do(ctx, "POST", path, requestBody, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Update(ctx context.Context, module, id string, data interface{}) (*SingleResponse, error) {
	path := fmt.Sprintf("/%s/%s", module, id)
	attrs, ok := data.(map[string]interface{})
	if !ok {
		attrs = map[string]interface{}{}
	}
	requestBody := BuildSingleWriteBody(module, id, "update", attrs)
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

func (c *Client) PostJSON(ctx context.Context, path string, body interface{}, result interface{}) error {
	c.logRequest(http.MethodPost, path)

	reqBody, err := prepareRequestBody(body)
	if err != nil {
		return err
	}

	reqURL := c.BaseURL + "/" + strings.TrimPrefix(path, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
	if c.Verbose >= 2 {
		headers := map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		}
		if c.AuthToken != "" {
			headers["Authorization"] = "Bearer " + c.AuthToken
		}
		fmt.Fprintf(os.Stderr, "[REQUEST HEADERS] %v\n", redactHeaders(headers))
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
		return &Error{
			Status:  resp.StatusCode,
			Body:    respBody,
			Message: string(respBody),
		}
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) GetSchema(ctx context.Context, module string) ([]byte, error) {
	path := "/schema/" + module
	c.logRequest(http.MethodGet, path)

	reqURL := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range c.requestHeaders(http.MethodGet) {
		req.Header.Set(key, value)
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logResponse(resp.StatusCode, respBody)

	if resp.StatusCode >= 400 {
		return nil, &Error{
			Status:  resp.StatusCode,
			Body:    respBody,
			Message: string(respBody),
		}
	}

	return respBody, nil
}
