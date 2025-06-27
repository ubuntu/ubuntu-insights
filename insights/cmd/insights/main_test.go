package main

import (
	"errors"
	"testing"
	"time"
)

type testApp struct {
	done            chan struct{}
	runError        bool
	userErrorReturn bool
}

func (a *testApp) Run() error {
	<-a.done
	if a.runError {
		return errors.New(("run error!"))
	}
	return nil
}

func (a testApp) UsageError() bool {
	return a.userErrorReturn
}

func (a testApp) Quit() {
	close(a.done)
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		runError   bool
		usageError bool

		wantReturnCode int
	}{
		"Run and exit successfully":                        {},
		"Run and exit error":                               {runError: true, wantReturnCode: 1},
		"Run and exit with usage error":                    {usageError: true, runError: true, wantReturnCode: 2},
		"Run and return with usage error but no run error": {usageError: true, wantReturnCode: 0},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := testApp{
				done:            make(chan struct{}),
				runError:        tc.runError,
				userErrorReturn: tc.usageError,
			}

			var rc int
			wait := make(chan struct{})

			go func() {
				rc = run(&a)
				close(wait)
			}()

			time.Sleep(100 * time.Millisecond)

			a.Quit()
			<-wait

			if rc != tc.wantReturnCode {
				t.Errorf("run() = %v, want %v", rc, tc.wantReturnCode)
			}
		})
	}
}
