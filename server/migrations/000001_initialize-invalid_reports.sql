-- +goose Up
CREATE TABLE invalid_reports (
    report_id UUID NOT NULL,
    entry_time TIMESTAMP NOT NULL,
    app_name TEXT NOT NULL,
    raw_report TEXT NOT NULL
);

CREATE INDEX idx_invalid_reports_report_id ON invalid_reports(report_id);
CREATE INDEX idx_invalid_reports_entry_time ON invalid_reports(entry_time);
CREATE INDEX idx_invalid_reports_app_name ON invalid_reports(app_name);

-- +goose Down
DROP INDEX IF EXISTS idx_invalid_reports_report_id;
DROP INDEX IF EXISTS idx_invalid_reports_entry_time;
DROP INDEX IF EXISTS idx_invalid_reports_app_name;

DROP TABLE IF EXISTS invalid_reports;
