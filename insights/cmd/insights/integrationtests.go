//go:build integrationtests

package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ubuntu/ubuntu-insights/insights/internal/collector/sysinfo"
	collectortestutils "github.com/ubuntu/ubuntu-insights/insights/internal/collector/testutils"
	"github.com/ubuntu/ubuntu-insights/insights/internal/constants"
	uploadertestutils "github.com/ubuntu/ubuntu-insights/insights/internal/uploader/testutils"
)

type MockTimeProvider struct {
	CurrentTime int64
}

func (m MockTimeProvider) Now() time.Time {
	return time.Unix(m.CurrentTime, 0)
}

// load any behavior modifiers from env variable.
func init() {
	if server_url := os.Getenv("UBUNTU_INSIGHTS_INTEGRATIONTESTS_SERVER_URL"); server_url != "" {
		uploadertestutils.SetServerURL(server_url)
	}

	if max_reports := os.Getenv("UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_REPORTS"); max_reports != "" {
		mr, err := strconv.ParseUint(max_reports, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("failed to parse UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_REPORTS: %v", err))
		}
		uploadertestutils.SetMaxReports(uint32(mr))
		collectortestutils.SetMaxReports(uint32(mr))
	}

	if time := os.Getenv("UBUNTU_INSIGHTS_INTEGRATIONTESTS_TIME"); time != "" {
		t, err := strconv.ParseInt(time, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("failed to parse UBUNTU_INSIGHTS_INTEGRATIONTESTS_TIME: %v", err))
		}
		uploadertestutils.SetTimeProvider(MockTimeProvider{CurrentTime: t})
		collectortestutils.SetTimeProvider(MockTimeProvider{CurrentTime: t})
	}

	if base_retry_period := os.Getenv("UBUNTU_INSIGHTS_INTEGRATIONTESTS_BASE_RETRY_PERIOD"); base_retry_period != "" {
		irp, err := time.ParseDuration(base_retry_period)
		if err != nil {
			panic(fmt.Sprintf("failed to parse UBUNTU_INSIGHTS_INTEGRATIONTESTS_BASE_RETRY_PERIOD: %v", err))
		}
		uploadertestutils.SetBaseRetryPeriod(irp)
	}

	if max_attempts := os.Getenv("UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_ATTEMPTS"); max_attempts != "" {
		ma, err := strconv.ParseUint(max_attempts, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("failed to parse UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_ATTEMPTS: %v", err))
		}
		uploadertestutils.SetMaxAttempts(uint32(ma))
	}

	if report_timeout := os.Getenv("UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_RETRY_PERIOD"); report_timeout != "" {
		rt, err := time.ParseDuration(report_timeout)
		if err != nil {
			panic(fmt.Sprintf("failed to parse UBUNTU_INSIGHTS_INTEGRATIONTESTS_MAX_RETRY_PERIOD: %v", err))
		}
		uploadertestutils.SetMaxRetryPeriod(rt)
	}

	if response_timeout := os.Getenv("UBUNTU_INSIGHTS_INTEGRATIONTESTS_RESPONSE_TIMEOUT"); response_timeout != "" {
		rt, err := time.ParseDuration(response_timeout)
		if err != nil {
			panic(fmt.Sprintf("failed to parse UBUNTU_INSIGHTS_INTEGRATIONTESTS_RESPONSE_TIMEOUT: %v", err))
		}
		uploadertestutils.SetResponseTimeout(rt)
	}

	if os.Getenv("UBUNTU_INSIGHTS_INTEGRATIONTESTS_SYSINFO") != "" {
		si := testSysInfo{}
		if sysinfoErr := os.Getenv("UBUNTU_INSIGHTS_INTEGRATIONTESTS_SYSINFO_ERR"); sysinfoErr != "" {
			si.err = fmt.Errorf("%s", sysinfoErr)
		}
		collectortestutils.SetSysInfo(si)
	}

	constants.Version = "Dev"
}

type testSysInfo struct {
	info sysinfo.Info
	err  error
}

func (m testSysInfo) Collect() (sysinfo.Info, error) {
	return m.info, m.err
}
