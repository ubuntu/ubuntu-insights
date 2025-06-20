package testutils

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

var (
	goCoverDir     string
	goCoverDirOnce sync.Once
)

// CoverDirEnv returns the cover dir env variable to run a go binary, if coverage is enabled.
func CoverDirEnv() string {
	if CoverDirForTests() == "" {
		return ""
	}
	return fmt.Sprintf("GOCOVERDIR=%s", CoverDirForTests())
}

// AppendCovEnv returns the env needed to enable coverage when running a go binary,
// if coverage is enabled.
func AppendCovEnv(env []string) []string {
	coverDir := CoverDirEnv()
	if coverDir == "" {
		return env
	}
	return append(env, coverDir)
}

// CoverDirForTests parses the test arguments and return the cover profile directory,
// if coverage is enabled.
func CoverDirForTests() string {
	goCoverDirOnce.Do(func() {
		if testing.CoverMode() == "" {
			return
		}

		for _, arg := range os.Args {
			if !strings.HasPrefix(arg, "-test.gocoverdir=") {
				continue
			}
			goCoverDir = strings.TrimPrefix(arg, "-test.gocoverdir=")
		}
	})

	return goCoverDir
}
