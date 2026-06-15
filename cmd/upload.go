package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"crmservice/internal/api"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

const uploadRequestTimeout = 60 * time.Second

func uploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload <module> <id> <file>",
		Short: "Upload a file and link it to an entity",
		Long: `Upload a file to the CRM files API and link it to an entity record.

Uses multipart/form-data with a "file" field and optional file metadata via --field.
Metadata fields must be scalar values (string, number, or boolean) and are validated
against the files module schema. Maximum file size is 25 MB.

Examples:
  crmservice upload accounts 123 ./contract.pdf
  crmservice upload entities 456 ./attachment.txt --field "file_usage_type=Entity Attachment"
  crmservice upload accounts 123 ./doc.pdf --field "file_access_type=Internal" --dry-run -o json`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUploadCommand(cmd, args[0], args[1], args[2])
		},
	}

	cmd.Flags().StringArray("field", []string{}, "File metadata attributes (validated against files module)")
	cmd.Flags().Bool("dry-run", false, "Show the upload request without sending it to the API")
	addCommonFlags(cmd, CommonFlagSet{Output: true, Verbose: true, Full: true})

	return cmd
}

func runUploadCommand(cmd *cobra.Command, module, id, filePath string) error {
	common, err := readCommonFlags(cmd)
	if err != nil {
		return err
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	attributes, err := getUploadAttributes(cmd)
	if err != nil {
		return err
	}

	cleanPath := filepath.Clean(filePath)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("file %s: %w", filePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("file %s is a directory", filePath)
	}
	if info.Size() > api.MaxUploadFileSize {
		return fmt.Errorf("file %s exceeds maximum upload size of 25 MB (got %d bytes)", filePath, info.Size())
	}

	apiPath := fmt.Sprintf("/%s/%s/files", module, id)

	url, err := getURLFromFlagOrEnv(cmd)
	if err != nil {
		return err
	}
	token, err := getTokenFromFlagEnvConfig(cmd)
	if err != nil {
		return err
	}
	if token != "" && len(attributes) > 0 {
		fields := make([]string, 0, len(attributes))
		for name := range attributes {
			fields = append(fields, name)
		}
		if err := validateModuleAttributes("files", fields, url, token, common.Verbose); err != nil {
			return output.ErrorResponse(err)
		}
	}

	if dryRun {
		return outputUploadDryRun(module, id, apiPath, cleanPath, info.Size(), attributes, common.OutputFormat)
	}

	token, err = getRequiredTokenFromFlagEnvConfig(cmd)
	if err != nil {
		return err
	}

	apiClient := newAPIClientWithTimeout(url, token, common.Verbose, uploadRequestTimeout)
	resp, err := apiClient.UploadFile(cmd.Context(), apiPath, api.FileUploadRequest{
		FilePath:   cleanPath,
		Attributes: attributes,
	})
	if err != nil {
		return output.ErrorResponse(err)
	}

	return output.ItemResponse(resp, output.Options{
		Format: common.OutputFormat,
		Full:   common.Full,
	})
}

func getUploadAttributes(cmd *cobra.Command) (map[string]interface{}, error) {
	fields, err := cmd.Flags().GetStringArray("field")
	if err != nil {
		return nil, err
	}

	attributes, err := parseFieldFlags(fields)
	if err != nil {
		return nil, err
	}
	for name, value := range attributes {
		if !isScalarUploadValue(value) {
			return nil, fmt.Errorf("invalid --field %q: upload metadata must be a scalar value (string, number, or boolean)", name+"="+formatScalarHint(value))
		}
	}
	return attributes, nil
}

func isScalarUploadValue(value interface{}) bool {
	switch value.(type) {
	case string, bool, json.Number, float64, float32, int, int64, int32, uint, uint64, uint32:
		return true
	default:
		return false
	}
}

func formatScalarHint(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func outputUploadDryRun(module, id, apiPath, filePath string, fileSize int64, attributes map[string]interface{}, outputFormat string) error {
	result := map[string]interface{}{
		"dry_run":   true,
		"operation": "upload",
		"module":    module,
		"id":        id,
		"path":      apiPath,
		"file":      filePath,
		"file_name": filepath.Base(filePath),
		"file_size": fileSize,
	}
	if len(attributes) > 0 {
		result["file_attributes"] = attributes
	}
	return output.ItemResponse(&api.SingleResponse{Data: result}, output.Options{
		Format: outputFormat,
		Full:   false,
	})
}
