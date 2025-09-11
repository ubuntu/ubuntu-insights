// TiCS: disabled // Test helpers.

package testutils

import (
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
)

var (
	isRace        bool
	isRaceOnce    sync.Once
	isVerbose     bool
	isVerboseOnce sync.Once
)

// IsVerbose returns whether the tests are running in verbose mode.
func IsVerbose() bool {
	isVerboseOnce.Do(func() {
		for _, arg := range os.Args {
			value, ok := strings.CutPrefix(arg, "-test.v=")
			if !ok {
				continue
			}
			isVerbose = value == "true"
		}
	})
	return isVerbose
}

func haveBuildFlag(flag string) bool {
	b, ok := debug.ReadBuildInfo()
	if !ok {
		panic("could not read build info")
	}

	flag = "-" + flag
	for _, s := range b.Settings {
		if s.Key != flag {
			continue
		}
		value, err := strconv.ParseBool(s.Value)
		if err != nil {
			panic(err)
		}
		return value
	}

	return false
}

// IsRace returns whether the tests are running with thread sanitizer.
func IsRace() bool {
	isRaceOnce.Do(func() { isRace = haveBuildFlag("race") })
	return isRace
}
