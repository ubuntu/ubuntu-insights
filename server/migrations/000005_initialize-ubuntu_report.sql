-- +goose Up
CREATE TABLE ubuntu_report (
    report_id UUID NOT NULL,
    entry_time TIMESTAMP NOT NULL,
    distribution TEXT NOT NULL,
    version TEXT NOT NULL,
    report JSONB,
    optout BOOLEAN NOT NULL
);

CREATE INDEX idx_ubuntu_report_report_id ON ubuntu_report(report_id);
CREATE INDEX idx_ubuntu_report_entry_time_optout ON ubuntu_report(entry_time, optout);
CREATE INDEX idx_ubuntu_report_distribution_version ON ubuntu_report(distribution, version);
CREATE INDEX idx_ubuntu_report_report ON ubuntu_report USING gin (report);

-- +goose Down
DROP INDEX IF EXISTS idx_ubuntu_report_entry_time_optout;
DROP INDEX IF EXISTS idx_ubuntu_report_distribution_version;
DROP INDEX IF EXISTS idx_ubuntu_report_report;
DROP INDEX IF EXISTS idx_ubuntu_report_report_id;

DROP TABLE IF EXISTS ubuntu_report;
