CREATE TABLE invalid_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL,
    entry_time TIMESTAMP NOT NULL,
    app_name TEXT NOT NULL,
    raw_report TEXT NOT NULL
);
