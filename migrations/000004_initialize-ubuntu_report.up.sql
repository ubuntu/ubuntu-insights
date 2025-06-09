CREATE TABLE ubuntu_report (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_time TIMESTAMP NOT NULL,
    distribution TEXT NOT NULL,
    version TEXT NOT NULL,
    report JSONB,
    optout BOOLEAN NOT NULL
);

CREATE INDEX idx_ubuntu_report_report ON ubuntu_report USING gin (report);
