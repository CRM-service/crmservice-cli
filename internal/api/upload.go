package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// MaxUploadFileSize matches the CRM REST API attachment limit (25 MiB).
const MaxUploadFileSize = 25 * 1024 * 1024

// FileUploadRequest describes a multipart file upload to the CRM API.
type FileUploadRequest struct {
	FilePath   string
	Attributes map[string]interface{}
}

func (c *Client) UploadFile(ctx context.Context, path string, req FileUploadRequest) (*SingleResponse, error) {
	body, contentType, err := buildMultipartUploadBody(req)
	if err != nil {
		return nil, err
	}

	c.logRequest(http.MethodPost, path)

	reqURL := c.BaseURL + "/" + strings.TrimPrefix(path, "/")
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, body)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Accept", "application/vnd.api+json")
	if c.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
	if c.Verbose >= 2 {
		headers := map[string]string{
			"Content-Type": contentType,
			"Accept":       "application/vnd.api+json",
		}
		if c.AuthToken != "" {
			headers["Authorization"] = "Bearer " + c.AuthToken
		}
		fmt.Fprintf(os.Stderr, "[REQUEST HEADERS] %v\n", redactHeaders(headers))
	}

	resp, err := c.HTTPClient.Do(httpReq)
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

	var result SingleResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func buildMultipartUploadBody(req FileUploadRequest) (io.Reader, string, error) {
	file, err := os.Open(req.FilePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, "", err
	}
	if info.IsDir() {
		return nil, "", fmt.Errorf("%s is a directory", req.FilePath)
	}
	if info.Size() > MaxUploadFileSize {
		return nil, "", fmt.Errorf("%s exceeds maximum upload size of 25 MB (got %d bytes)", req.FilePath, info.Size())
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(req.FilePath))
	if err != nil {
		return nil, "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, "", err
	}

	for key, value := range req.Attributes {
		if err := writer.WriteField(fmt.Sprintf("attributes[%s]", key), formatUploadAttributeValue(value)); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}

func formatUploadAttributeValue(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case json.Number:
		return typed.String()
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", typed), "0"), ".")
	default:
		return fmt.Sprintf("%v", typed)
	}
}
