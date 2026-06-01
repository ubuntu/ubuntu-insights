-- +goose Up
CREATE TABLE windows (
    report_id UUID NOT NULL,
    entry_time TIMESTAMP NOT NULL,
    insights_version TEXT,
    collection_time TIMESTAMP,
    hardware JSONB,
    software JSONB,
    platform JSONB,
    source_metrics JSONB,
    optout BOOLEAN NOT NULL
);

CREATE INDEX idx_windows_report_id ON windows(report_id);
CREATE INDEX idx_windows_entry_time_optout ON windows(entry_time, optout);
CREATE INDEX idx_windows_collection_time ON windows(collection_time);
CREATE INDEX idx_windows_hardware ON windows USING gin (hardware);
CREATE INDEX idx_windows_software ON windows USING gin (software);
CREATE INDEX idx_windows_platform ON windows USING gin (platform);
CREATE INDEX idx_windows_source_metrics ON windows USING gin (source_metrics);

-- +goose Down
DROP INDEX IF EXISTS idx_windows_entry_time_optout;
DROP INDEX IF EXISTS idx_windows_source_metrics;
DROP INDEX IF EXISTS idx_windows_platform;
DROP INDEX IF EXISTS idx_windows_software;
DROP INDEX IF EXISTS idx_windows_hardware;
DROP INDEX IF EXISTS idx_windows_collection_time;
DROP INDEX IF EXISTS idx_windows_report_id;

DROP TABLE IF EXISTS windows;
