package consent_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofrs/flock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/ubuntu-insights/internal/consent"
	"github.com/ubuntu/ubuntu-insights/internal/testutils"
)

type lockType uint

const (
	noLock lockType = iota
	readLock
	writeLock
)

// consentDir is a struct that holds a test's temporary directory and its locks.
// It should be cleaned up after the test is done.
type consentDir struct {
	dir         string
	sourceLocks map[string]*flock.Flock
	globalLock  *flock.Flock
}

func TestGetConsentStates(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		sources     []string
		lockSources map[string]lockType
		globalFile  string
		lockGlobal  lockType

		wantErr bool
	}{
		"No Global File, No Locks": {},

		// Global File Tests
		"Valid True Global File, No Locks":          {globalFile: "valid_true-consent.toml"},
		"Valid False Global File, No Locks":         {globalFile: "valid_false-consent.toml"},
		"Valid True Global File, Global Read Lock":  {globalFile: "valid_true-consent.toml", lockGlobal: readLock},
		"Valid True Global File, Global Write Lock": {globalFile: "valid_true-consent.toml", lockGlobal: writeLock},
		"Invalid Value Global File, No Locks":       {globalFile: "invalid_value-consent.toml"},
		"Invalid File Global File, No Locks":        {globalFile: "invalid_file-consent.toml"},

		// Lock sources
		"Valid True Global File, Source Read Lock":                         {globalFile: "valid_true-consent.toml", lockSources: map[string]lockType{"valid_true": readLock}},
		"Valid True Global File, Source Write Lock":                        {globalFile: "valid_true-consent.toml", lockSources: map[string]lockType{"valid_true": writeLock}},
		"Valid True Global File, Source Read Lock, Global Read Lock":       {globalFile: "valid_true-consent.toml", lockSources: map[string]lockType{"valid_true": readLock}, lockGlobal: readLock},
		"Valid True Global File, Source Write Lock, Global Read Lock":      {globalFile: "valid_true-consent.toml", lockSources: map[string]lockType{"valid_true": writeLock}, lockGlobal: readLock},
		"Valid True Global File, Dual Source Write Lock, Global Read Lock": {globalFile: "valid_true-consent.toml", lockSources: map[string]lockType{"valid_true": writeLock, "invalid_value": writeLock}, lockGlobal: readLock},

		// Source Specific Tests
		"Valid True Global File, No Locks, Valid True Source, No Locks":                       {globalFile: "valid_true-consent.toml", sources: []string{"valid_true"}},
		"Valid True Global File, No Locks, Valid False Source, No Locks":                      {globalFile: "valid_true-consent.toml", sources: []string{"valid_false"}},
		"Valid True Global File, No Locks, Invalid Value Source, No Locks":                    {globalFile: "valid_true-consent.toml", sources: []string{"invalid_value"}},
		"Valid True Global File, No Locks, Invalid File Source, No Locks":                     {globalFile: "valid_true-consent.toml", sources: []string{"invalid_file"}},
		"Valid True Global File, No Locks, No File Source, No Locks":                          {globalFile: "valid_true-consent.toml", sources: []string{"not_a_file"}},
		"Valid True Global File, No Locks, 2 Multiple Sources (VTrue, VFalse), No Locks":      {globalFile: "valid_true-consent.toml", sources: []string{"valid_true", "valid_false"}},
		"Valid True Global File, No Locks, 3 Multiple Sources (VTrue, VFalse, NAF), No Locks": {globalFile: "valid_true-consent.toml", sources: []string{"valid_true", "valid_false", "not_a_file"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cdir, err := setupTmpConsentFiles(t, tc.lockSources, tc.globalFile, tc.lockGlobal)
			require.NoError(t, err, "failed to setup temporary consent files")
			defer cdir.cleanup(t)
			cm := consent.New(cdir.dir)

			got, err := cm.GetConsentStates(tc.sources)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetConsentStates should return expected consent states")
		})
	}
}

func TestSetConsentStates(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		sources       []string
		consentStates map[string]bool
		lockSources   map[string]lockType
		globalFile    string
		lockGlobal    lockType

		writeSource string
		writeState  bool

		wantErr bool
	}{
		// New File Tests No Locks
		"New File, No Locks, Write Global False": {},
		"New File, No Locks, Write Global True":  {writeState: true},
		"New File, No Locks, Write Source True":  {writeSource: "new_true", writeState: true},
		"New File, No Locks, Write Source False": {writeSource: "new_false"},

		// Overwrite File, No Locks, Different State
		"Overwrite File, No Locks, Write Diff Global False": {globalFile: "valid_true-consent.toml", writeState: false},
		"Overwrite File, No Locks, Write Diff Global True":  {globalFile: "valid_false-consent.toml", writeState: true},
		"Overwrite File, No Locks, Write Diff Source True":  {globalFile: "valid_true-consent.toml", writeSource: "valid_false", writeState: true},
		"Overwrite File, No Locks, Write Diff Source False": {globalFile: "valid_true-consent.toml", writeSource: "valid_true", writeState: false},

		// Overwrite File, No Locks, Same State
		"Overwrite File, No Locks, Write Global True":  {globalFile: "valid_true-consent.toml", writeState: true},
		"Overwrite File, No Locks, Write Global False": {globalFile: "valid_false-consent.toml", writeState: false},
		"Overwrite File, No Locks, Write Source True":  {globalFile: "valid_true-consent.toml", writeSource: "valid_true", writeState: true},
		"Overwrite File, No Locks, Write Source False": {globalFile: "valid_false-consent.toml", writeSource: "valid_false"},

		// Overwrite File, Read Locks, Different State
		"Overwrite File, Source Read Lock, Write Global False": {globalFile: "valid_true-consent.toml", lockGlobal: readLock, writeState: false, wantErr: true},
		"Overwrite File, Source Read Lock, Write Global True":  {globalFile: "valid_false-consent.toml", lockGlobal: readLock, writeState: true, wantErr: true},
		"Overwrite File, Source Read Lock, Write Source True":  {globalFile: "valid_true-consent.toml", lockSources: map[string]lockType{"valid_false": readLock}, writeSource: "valid_false", writeState: true, wantErr: true},
		"Overwrite File, Source Read Lock, Write Source False": {globalFile: "valid_false-consent.toml", lockSources: map[string]lockType{"valid_true": readLock}, writeSource: "valid_true", writeState: false, wantErr: true},

		// Overwrite File, Write Locks, Different State
		"Overwrite File, Source Write Lock, Write Global False": {globalFile: "valid_true-consent.toml", lockGlobal: writeLock, writeState: false, wantErr: true},
		"Overwrite File, Source Write Lock, Write Global True":  {globalFile: "valid_false-consent.toml", lockGlobal: writeLock, writeState: true, wantErr: true},
		"Overwrite File, Source Write Lock, Write Source True":  {globalFile: "valid_true-consent.toml", lockSources: map[string]lockType{"valid_false": writeLock}, writeSource: "valid_false", writeState: true, wantErr: true},
		"Overwrite File, Source Write Lock, Write Source False": {globalFile: "valid_false-consent.toml", lockSources: map[string]lockType{"valid_true": writeLock}, writeSource: "valid_true", writeState: false, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cdir, err := setupTmpConsentFiles(t, tc.lockSources, tc.globalFile, tc.lockGlobal)
			require.NoError(t, err, "failed to setup temporary consent files")
			defer cdir.cleanup(t)
			cm := consent.New(cdir.dir)

			err = cm.SetConsentState(tc.writeSource, tc.writeState)
			if tc.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}
			require.NoError(t, err, "got an unexpected error")

			got, err := cm.GetConsentStates(tc.sources)
			require.NoError(t, err, "got an unexpected error while getting consent states")

			want := testutils.LoadWithUpdateFromGoldenYAML(t, got)
			require.Equal(t, want, got, "GetConsentStates should return expected consent states")
		})
	}
}

// cleanup unlocks all the locks and removes the temporary directory including its contents.
func (cdir consentDir) cleanup(t *testing.T) {
	t.Helper()
	for i := range cdir.sourceLocks {
		assert.NoError(t, cdir.sourceLocks[i].Unlock(), "failed to unlock source lock")
	}

	if cdir.globalLock != nil {
		assert.NoError(t, cdir.globalLock.Unlock(), "failed to unlock global lock")
	}

	assert.NoError(t, os.RemoveAll(cdir.dir), "failed to remove temporary directory")
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func copyDir(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath)
	})
}

func setupTmpConsentFiles(t *testing.T, lockSources map[string]lockType, globalFile string, lockGlobal lockType) (*consentDir, error) {
	t.Helper()
	cdir := consentDir{sourceLocks: make(map[string]*flock.Flock)}

	// Setup temporary directory
	var err error
	cdir.dir, err = os.MkdirTemp("", "consent-files")
	if err != nil {
		return &cdir, fmt.Errorf("failed to create temporary directory: %v", err)
	}

	if err = copyDir(filepath.Join("testdata", "consent_files"), cdir.dir); err != nil {
		return &cdir, fmt.Errorf("failed to copy testdata directory to temporary directory: %v", err)
	}

	// Setup globalfile if provided
	if globalFile != "" {
		if err = copyFile(filepath.Join(cdir.dir, globalFile), filepath.Join(cdir.dir, "consent.toml")); err != nil {
			return &cdir, fmt.Errorf("failed to copy requested global consent file: %v", err)
		}
	}

	// Setup lock files
	for source, lock := range lockSources {
		if lock == noLock {
			continue
		}

		lockPath := filepath.Join(cdir.dir, source+"-consent.toml.lock")
		cdir.sourceLocks[source] = flock.New(lockPath)
		switch lock {
		case readLock:
			err = cdir.sourceLocks[source].RLock()
		case writeLock:
			err = cdir.sourceLocks[source].Lock()
		}

		if err != nil {
			return &cdir, fmt.Errorf("failed to acquire lock on consent file for source %s: %v", source, err)
		}
	}

	// Setup global lock file
	if lockGlobal != noLock {
		if globalFile == "" {
			return &cdir, fmt.Errorf("global file must be provided if global lock is requested")
		}

		lockPath := filepath.Join(cdir.dir, "consent.toml.lock")
		cdir.globalLock = flock.New(lockPath)
		switch lockGlobal {
		case readLock:
			err = cdir.globalLock.RLock()
		case writeLock:
			err = cdir.globalLock.Lock()
		}

		if err != nil {
			return &cdir, fmt.Errorf("failed to acquire lock on global consent file: %v", err)
		}
	}

	return &cdir, nil
}
