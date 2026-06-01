-- Sample application data for integration testing.
-- Loaded after the golang-migrate dump to verify that existing data is preserved
-- through the migration tooling transition from golang-migrate to goose.

INSERT INTO invalid_reports (report_id, entry_time, app_name, raw_report)
VALUES ('11111111-1111-1111-1111-111111111111', '2025-01-15 10:00:00', 'test-app', '{"invalid": true}');

INSERT INTO linux (report_id, entry_time, insights_version, collection_time, hardware, software, platform, source_metrics, optout)
VALUES ('22222222-2222-2222-2222-222222222222', '2025-02-20 12:00:00', '1.0.0', '2025-02-20 11:55:00', '{"cpu": "x86_64", "memory": "16GB"}', '{"os": "Ubuntu 24.04"}', '{"arch": "amd64"}', '{"source": "apt"}', false);

INSERT INTO windows (report_id, entry_time, insights_version, collection_time, hardware, software, platform, source_metrics, optout)
VALUES ('33333333-3333-3333-3333-333333333333', '2025-03-01 09:00:00', '1.0.0', '2025-03-01 08:55:00', '{"cpu": "x86_64"}', '{"os": "Windows 11"}', '{"arch": "amd64"}', '{"source": "winget"}', false);

INSERT INTO darwin (report_id, entry_time, insights_version, collection_time, hardware, software, platform, source_metrics, optout)
VALUES ('44444444-4444-4444-4444-444444444444', '2025-03-05 14:00:00', '1.0.0', '2025-03-05 13:55:00', '{"cpu": "arm64"}', '{"os": "macOS 15"}', '{"arch": "arm64"}', '{"source": "brew"}', false);

INSERT INTO ubuntu_report (report_id, entry_time, distribution, version, report, optout)
VALUES ('55555555-5555-5555-5555-555555555555', '2025-04-01 08:00:00', 'Ubuntu', '24.04', '{"legacy": true}', false);

INSERT INTO ubuntu_desktop_provision (report_id, entry_time, insights_version, collection_time, hardware, software, platform, source_metrics, optout)
VALUES ('66666666-6666-6666-6666-666666666666', '2025-04-10 16:00:00', '1.0.0', '2025-04-10 15:55:00', '{"cpu": "x86_64"}', '{"os": "Ubuntu 24.04"}', '{"arch": "amd64"}', '{"source": "oem"}', true);

INSERT INTO ubuntu_release_upgrader (report_id, entry_time, insights_version, collection_time, hardware, software, platform, source_metrics, optout)
VALUES ('77777777-7777-7777-7777-777777777777', '2025-05-01 11:00:00', '1.0.0', '2025-05-01 10:55:00', '{"cpu": "x86_64"}', '{"os": "Ubuntu 24.04"}', '{"arch": "amd64"}', '{"source": "do-release-upgrade"}', false);

INSERT INTO wsl_setup (report_id, entry_time, insights_version, collection_time, hardware, software, platform, source_metrics, optout)
VALUES ('88888888-8888-8888-8888-888888888888', '2025-05-15 09:00:00', '1.0.0', '2025-05-15 08:55:00', '{"cpu": "x86_64"}', '{"os": "Ubuntu 24.04"}', '{"arch": "amd64"}', '{"source": "wsl"}', false);
