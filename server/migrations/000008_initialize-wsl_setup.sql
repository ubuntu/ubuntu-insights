-- +goose Up
CREATE TABLE wsl_setup (
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

CREATE INDEX idx_wsl_setup_report_id ON wsl_setup(report_id);
CREATE INDEX idx_wsl_setup_entry_time_optout ON wsl_setup(entry_time, optout);
CREATE INDEX idx_wsl_setup_collection_time ON wsl_setup(collection_time);
CREATE INDEX idx_wsl_setup_hardware ON wsl_setup USING gin (hardware);
CREATE INDEX idx_wsl_setup_software ON wsl_setup USING gin (software);
CREATE INDEX idx_wsl_setup_platform ON wsl_setup USING gin (platform);
CREATE INDEX idx_wsl_setup_source_metrics ON wsl_setup USING gin (source_metrics);

-- +goose Down
DROP INDEX IF EXISTS idx_wsl_setup_entry_time_optout;
DROP INDEX IF EXISTS idx_wsl_setup_source_metrics;
DROP INDEX IF EXISTS idx_wsl_setup_platform;
DROP INDEX IF EXISTS idx_wsl_setup_software;
DROP INDEX IF EXISTS idx_wsl_setup_hardware;
DROP INDEX IF EXISTS idx_wsl_setup_collection_time;
DROP INDEX IF EXISTS idx_wsl_setup_report_id;

DROP TABLE IF EXISTS wsl_setup;
