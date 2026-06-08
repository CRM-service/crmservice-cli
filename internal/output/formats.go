package output

import "fmt"

// OutputFormats is the canonical list of supported CLI output formats.
var OutputFormats = []string{"table", "json", "yaml", "jsonl", "csv"}

func ValidOutputFormat(format string) bool {
	for _, allowed := range OutputFormats {
		if format == allowed {
			return true
		}
	}
	return false
}

func FormatsHelp() string {
	if len(OutputFormats) == 0 {
		return ""
	}
	help := OutputFormats[0]
	for _, format := range OutputFormats[1:] {
		help += ", " + format
	}
	return help
}

func ValidateOutputFormat(format string) error {
	if ValidOutputFormat(format) {
		return nil
	}
	return fmt.Errorf("invalid output format: %s. Valid formats: %s", format, FormatsHelp())
}
