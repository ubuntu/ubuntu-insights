package handlers

// ReportsDir returns the directory where reports are stored.
func (u *Upload) ReportsDir() string {
	return u.jsonHandler.reportsDir
}

func (u *LegacyReport) ReportsDir() string {
	return u.jsonHandler.reportsDir
}
