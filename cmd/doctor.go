package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"crmservice/internal/config"
	"crmservice/internal/output"

	"github.com/spf13/cobra"
)

var doctorColumns = []string{
	"version", "config_path", "config_present", "api_url", "api_url_configured",
	"api_reachable", "token_present", "authenticated", "cache_dir", "cache_writable", "ok",
}

var (
	doctorCheckAPIReachable = checkAPIReachable
	doctorFetchCurrentUser  = fetchCurrentUser
)

func doctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run preflight checks for agent and automation setups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat, err := getOutputFormatFromFlagConfig(cmd)
			if err != nil {
				return err
			}

			result := runDoctorChecks(cmd)
			if err := outputDoctor(outputFormat, result); err != nil {
				return err
			}
			passed, hasOK := result["ok"].(bool)
			if !hasOK || !passed {
				issueCount := 0
				if issues, hasIssues := result["issues"].([]string); hasIssues {
					issueCount = len(issues)
				}
				return &output.ReportedError{Err: fmt.Errorf("doctor: %d check(s) failed", issueCount)}
			}
			return nil
		},
	}

	addCommonFlags(cmd, CommonFlagSet{Output: true})

	return cmd
}

func runDoctorChecks(cmd *cobra.Command) map[string]interface{} {
	issues := []string{}

	configPath := config.DefaultConfigPath()
	if configFile != "" {
		configPath = configFile
	}
	configPresent := false
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			configPresent = true
		}
	}

	url, err := getOptionalURLFromFlagEnvConfig(cmd)
	if err != nil {
		issues = append(issues, "api_url_error")
	}
	apiURLConfigured := url != ""
	if !apiURLConfigured {
		issues = append(issues, "api_url_missing")
	}

	apiReachable := false
	if apiURLConfigured {
		apiReachable = doctorCheckAPIReachable(cmd.Context(), url)
		if !apiReachable {
			issues = append(issues, "api_unreachable")
		}
	}

	token, err := getTokenFromFlagEnvConfig(cmd)
	if err != nil {
		issues = append(issues, "token_error")
	}
	tokenPresent := token != ""
	if !tokenPresent {
		issues = append(issues, "token_missing")
	}

	authenticated := false
	var user map[string]interface{}
	if apiURLConfigured && tokenPresent && apiReachable {
		fetchedUser, err := doctorFetchCurrentUser(cmd.Context(), url, token)
		switch {
		case err == nil:
			authenticated = true
			user = fetchedUser
		case isUnauthorizedError(err):
			issues = append(issues, "token_invalid")
		default:
			issues = append(issues, "auth_check_failed")
		}
	}

	cacheDir := ""
	cacheWritable := false
	if apiURLConfigured {
		cacheDir = getSchemaCacheDir(url)
		cacheWritable = checkCacheWritable(cacheDir)
		if !cacheWritable {
			issues = append(issues, "cache_not_writable")
		}
	}

	result := map[string]interface{}{
		"version":            versionString(),
		"config_path":        configPath,
		"config_present":     configPresent,
		"api_url":            url,
		"api_url_configured": apiURLConfigured,
		"api_reachable":      apiReachable,
		"token_present":      tokenPresent,
		"authenticated":      authenticated,
		"cache_dir":          cacheDir,
		"cache_writable":     cacheWritable,
		"ok":                 len(issues) == 0,
		"issues":             issues,
	}
	if user != nil {
		result["user"] = user
	}
	return result
}

func checkAPIReachable(ctx context.Context, url string) bool {
	client := &http.Client{Timeout: getTimeoutFromConfig()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Accept", "application/vnd.api+json")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode < 500
}

func checkCacheWritable(cacheDir string) bool {
	if cacheDir == "" {
		return false
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return false
	}
	testFile := filepath.Join(cacheDir, ".doctor-write-test")
	if err := os.WriteFile(testFile, []byte("ok"), 0o600); err != nil {
		return false
	}
	_ = os.Remove(testFile)
	return true
}

func outputDoctor(format string, data map[string]interface{}) error {
	return output.WriteStructured(format, data, output.StructuredOptions{
		Columns:      doctorColumns,
		ExtraColumns: []string{"issues", "user"},
	})
}
