package osutil

import "os"

// UserHomeDir is overridden in tests to control home-directory resolution.
var UserHomeDir = os.UserHomeDir
