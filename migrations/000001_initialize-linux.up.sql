CREATE TABLE linux (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_time TIMESTAMP NOT NULL,
    insights_version TEXT,
    collection_time TIMESTAMP,
    hardware JSONB,
    software JSONB,
    platform JSONB,
    source_metrics JSONB,
    optout BOOLEAN NOT NULL
);

CREATE INDEX idx_linux_hardware ON linux USING gin (hardware);
CREATE INDEX idx_linux_software ON linux USING gin (software);
CREATE INDEX idx_linux_platform ON linux USING gin (platform);
CREATE INDEX idx_linux_source_metrics ON linux USING gin (source_metrics);
