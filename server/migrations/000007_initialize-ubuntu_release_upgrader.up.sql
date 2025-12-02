CREATE TABLE ubuntu_release_upgrader (
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

CREATE INDEX idx_ubuntu_release_upgrader_report_id ON ubuntu_release_upgrader(report_id);
CREATE INDEX idx_ubuntu_release_upgrader_entry_time_optout ON ubuntu_release_upgrader(entry_time, optout);
CREATE INDEX idx_ubuntu_release_upgrader_collection_time ON ubuntu_release_upgrader(collection_time);
CREATE INDEX idx_ubuntu_release_upgrader_hardware ON ubuntu_release_upgrader USING gin (hardware);
CREATE INDEX idx_ubuntu_release_upgrader_software ON ubuntu_release_upgrader USING gin (software);
CREATE INDEX idx_ubuntu_release_upgrader_platform ON ubuntu_release_upgrader USING gin (platform);
CREATE INDEX idx_ubuntu_release_upgrader_source_metrics ON ubuntu_release_upgrader USING gin (source_metrics);
