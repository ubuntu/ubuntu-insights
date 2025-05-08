CREATE TABLE darwin (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_time TIMESTAMP NOT NULL,
    insights_version TEXT,
    hardware JSONB,
    software JSONB,
    platform JSONB,
    source_metrics JSONB,
    optout BOOLEAN NOT NULL
);

CREATE INDEX idx_darwin_hardware ON darwin USING gin (hardware);
CREATE INDEX idx_darwin_software ON darwin USING gin (software);
CREATE INDEX idx_darwin_platform ON darwin USING gin (platform);
CREATE INDEX idx_darwin_source_metrics ON darwin USING gin (source_metrics);
