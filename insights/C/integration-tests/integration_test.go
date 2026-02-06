//go:build !(darwin && amd64)

// These tests run into "fatal error: addspecial on invalid pointer" on macOS AMD64.
// The underlying cause is a bit beyond me at this point, so we just skip the tests there for now.
//
// Example logs:
// https://productionresultssa0.blob.core.windows.net/actions-results/fa75e102-de88-4d46-bf3d-413a071f6ce8/workflow-job-run-eff1d76a-5e64-5e8f-bebb-defc68243e9a/logs/job/job-logs.txt?rsct=text%2Fplain&se=2026-02-06T03%3A34%3A14Z&sig=NAwzDm9SGQbVbLi5NM%2Fp%2Ba0bruGGdtfloPJhXYT4Ujc%3D&ske=2026-02-06T05%3A51%3A48Z&skoid=ca7593d4-ee42-46cd-af88-8b886a2f84eb&sks=b&skt=2026-02-06T01%3A51%3A48Z&sktid=398a6654-997b-47e9-b12b-9515b896b4de&skv=2025-11-05&sp=r&spr=https&sr=b&st=2026-02-06T03%3A24%3A09Z&sv=2025-11-05
//
// https://productionresultssa0.blob.core.windows.net/actions-results/fa75e102-de88-4d46-bf3d-413a071f6ce8/workflow-job-run-11f9373d-f0bc-548b-aeae-df965c53cd11/logs/job/job-logs.txt?rsct=text%2Fplain&se=2026-02-06T03%3A22%3A00Z&sig=dGPIQjp71mnpaNuMGq7381DQUybEK4BmhJZo6p6anDY%3D&ske=2026-02-06T05%3A59%3A30Z&skoid=ca7593d4-ee42-46cd-af88-8b886a2f84eb&sks=b&skt=2026-02-06T01%3A59%3A30Z&sktid=398a6654-997b-47e9-b12b-9515b896b4de&skv=2025-11-05&sp=r&spr=https&sr=b&st=2026-02-06T03%3A11%3A55Z&sv=2025-11-05

package libinsights_test

import (
	"os"
	"testing"

	libinsights "github.com/ubuntu/ubuntu-insights/insights/C/integration-tests"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestCollect(t *testing.T) {
	libinsights.TestCollect(t)
}

func TestCompileAndWrite(t *testing.T) {
	libinsights.TestCompileAndWrite(t)
}

func TestUpload(t *testing.T) {
	libinsights.TestUpload(t)
}

func TestSetConsent(t *testing.T) {
	libinsights.TestSetConsent(t)
}

func TestGetConsent(t *testing.T) {
	libinsights.TestGetConsent(t)
}

func TestCallback(t *testing.T) {
	libinsights.TestCallback(t)
}

// Wrapper to facilitate log capture in set log callback tests.
func TestHelperCallbackWorkWrapper(t *testing.T) {
	libinsights.TestHelperCallbackWork(t)
}
