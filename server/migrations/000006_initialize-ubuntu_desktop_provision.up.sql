CREATE TABLE ubuntu_desktop_provision (
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

CREATE INDEX idx_ubuntu_desktop_provision_report_id ON ubuntu_desktop_provision(report_id);
CREATE INDEX idx_ubuntu_desktop_provision_entry_time_optout ON ubuntu_desktop_provision(entry_time, optout);
CREATE INDEX idx_ubuntu_desktop_provision_collection_time ON ubuntu_desktop_provision(collection_time);
CREATE INDEX idx_ubuntu_desktop_provision_hardware ON ubuntu_desktop_provision USING gin (hardware);
CREATE INDEX idx_ubuntu_desktop_provision_software ON ubuntu_desktop_provision USING gin (software);
CREATE INDEX idx_ubuntu_desktop_provision_platform ON ubuntu_desktop_provision USING gin (platform);
CREATE INDEX idx_ubuntu_desktop_provision_source_metrics ON ubuntu_desktop_provision USING gin (source_metrics);
