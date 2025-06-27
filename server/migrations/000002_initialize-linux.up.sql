CREATE TABLE linux (
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

CREATE INDEX idx_linux_report_id ON linux(report_id);
CREATE INDEX idx_linux_entry_time_optout ON linux(entry_time, optout);
CREATE INDEX idx_linux_collection_time ON linux(collection_time);
CREATE INDEX idx_linux_hardware ON linux USING gin (hardware);
CREATE INDEX idx_linux_software ON linux USING gin (software);
CREATE INDEX idx_linux_platform ON linux USING gin (platform);
CREATE INDEX idx_linux_source_metrics ON linux USING gin (source_metrics);
