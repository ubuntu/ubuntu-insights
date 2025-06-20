package report

type ReportStash = reportStash

// ReportStash returns the reportStash of the report.
func (r Report) ReportStash() ReportStash {
	return r.reportStash
}
