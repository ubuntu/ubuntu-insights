package report

type ReportStash = reportStash

// ReportStash returns the reportStash of the report.
//
//nolint:revive // This is a false positive as we returned a typed alias and not the private type.
func (r Report) ReportStash() ReportStash {
	return r.reportStash
}
