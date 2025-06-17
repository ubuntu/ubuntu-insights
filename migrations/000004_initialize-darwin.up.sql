CREATE TABLE darwin (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

CREATE INDEX idx_darwin_report_id ON darwin(report_id);
CREATE INDEX idx_darwin_entry_time ON darwin(entry_time);
CREATE INDEX idx_darwin_collection_time ON darwin(collection_time);
CREATE INDEX idx_darwin_hardware ON darwin USING gin (hardware);
CREATE INDEX idx_darwin_software ON darwin USING gin (software);
CREATE INDEX idx_darwin_platform ON darwin USING gin (platform);
CREATE INDEX idx_darwin_source_metrics ON darwin USING gin (source_metrics);
CREATE INDEX idx_darwin_optout ON darwin(optout);
