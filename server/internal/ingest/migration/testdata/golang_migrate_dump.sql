-- This file is an exact pg_dump of a database after golang-migrate applied all 8
-- migrations (version 8, dirty=false). It was generated from a real Postgres 17
-- instance managed by golang-migrate v4.19.1.
--
-- Generated with:
--   pg_dump --no-owner --no-privileges --no-comments --inserts
--
-- Purpose: used as an integration test fixture to validate the goose bootstrap
-- logic against the exact schema that golang-migrate produces.
--
-- DO NOT EDIT MANUALLY. To regenerate, apply golang-migrate migrations to a fresh
-- Postgres database and re-dump.

CREATE TABLE public.darwin (
    report_id uuid NOT NULL,
    entry_time timestamp without time zone NOT NULL,
    insights_version text,
    collection_time timestamp without time zone,
    hardware jsonb,
    software jsonb,
    platform jsonb,
    source_metrics jsonb,
    optout boolean NOT NULL
);
CREATE TABLE public.invalid_reports (
    report_id uuid NOT NULL,
    entry_time timestamp without time zone NOT NULL,
    app_name text NOT NULL,
    raw_report text NOT NULL
);
CREATE TABLE public.linux (
    report_id uuid NOT NULL,
    entry_time timestamp without time zone NOT NULL,
    insights_version text,
    collection_time timestamp without time zone,
    hardware jsonb,
    software jsonb,
    platform jsonb,
    source_metrics jsonb,
    optout boolean NOT NULL
);
CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);
CREATE TABLE public.ubuntu_desktop_provision (
    report_id uuid NOT NULL,
    entry_time timestamp without time zone NOT NULL,
    insights_version text,
    collection_time timestamp without time zone,
    hardware jsonb,
    software jsonb,
    platform jsonb,
    source_metrics jsonb,
    optout boolean NOT NULL
);
CREATE TABLE public.ubuntu_release_upgrader (
    report_id uuid NOT NULL,
    entry_time timestamp without time zone NOT NULL,
    insights_version text,
    collection_time timestamp without time zone,
    hardware jsonb,
    software jsonb,
    platform jsonb,
    source_metrics jsonb,
    optout boolean NOT NULL
);
CREATE TABLE public.ubuntu_report (
    report_id uuid NOT NULL,
    entry_time timestamp without time zone NOT NULL,
    distribution text NOT NULL,
    version text NOT NULL,
    report jsonb,
    optout boolean NOT NULL
);
CREATE TABLE public.windows (
    report_id uuid NOT NULL,
    entry_time timestamp without time zone NOT NULL,
    insights_version text,
    collection_time timestamp without time zone,
    hardware jsonb,
    software jsonb,
    platform jsonb,
    source_metrics jsonb,
    optout boolean NOT NULL
);
CREATE TABLE public.wsl_setup (
    report_id uuid NOT NULL,
    entry_time timestamp without time zone NOT NULL,
    insights_version text,
    collection_time timestamp without time zone,
    hardware jsonb,
    software jsonb,
    platform jsonb,
    source_metrics jsonb,
    optout boolean NOT NULL
);
INSERT INTO public.schema_migrations VALUES (8, false);
ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);
CREATE INDEX idx_darwin_collection_time ON public.darwin USING btree (collection_time);
CREATE INDEX idx_darwin_entry_time_optout ON public.darwin USING btree (entry_time, optout);
CREATE INDEX idx_darwin_hardware ON public.darwin USING gin (hardware);
CREATE INDEX idx_darwin_platform ON public.darwin USING gin (platform);
CREATE INDEX idx_darwin_report_id ON public.darwin USING btree (report_id);
CREATE INDEX idx_darwin_software ON public.darwin USING gin (software);
CREATE INDEX idx_darwin_source_metrics ON public.darwin USING gin (source_metrics);
CREATE INDEX idx_invalid_reports_app_name ON public.invalid_reports USING btree (app_name);
CREATE INDEX idx_invalid_reports_entry_time ON public.invalid_reports USING btree (entry_time);
CREATE INDEX idx_invalid_reports_report_id ON public.invalid_reports USING btree (report_id);
CREATE INDEX idx_linux_collection_time ON public.linux USING btree (collection_time);
CREATE INDEX idx_linux_entry_time_optout ON public.linux USING btree (entry_time, optout);
CREATE INDEX idx_linux_hardware ON public.linux USING gin (hardware);
CREATE INDEX idx_linux_platform ON public.linux USING gin (platform);
CREATE INDEX idx_linux_report_id ON public.linux USING btree (report_id);
CREATE INDEX idx_linux_software ON public.linux USING gin (software);
CREATE INDEX idx_linux_source_metrics ON public.linux USING gin (source_metrics);
CREATE INDEX idx_ubuntu_desktop_provision_collection_time ON public.ubuntu_desktop_provision USING btree (collection_time);
CREATE INDEX idx_ubuntu_desktop_provision_entry_time_optout ON public.ubuntu_desktop_provision USING btree (entry_time, optout);
CREATE INDEX idx_ubuntu_desktop_provision_hardware ON public.ubuntu_desktop_provision USING gin (hardware);
CREATE INDEX idx_ubuntu_desktop_provision_platform ON public.ubuntu_desktop_provision USING gin (platform);
CREATE INDEX idx_ubuntu_desktop_provision_report_id ON public.ubuntu_desktop_provision USING btree (report_id);
CREATE INDEX idx_ubuntu_desktop_provision_software ON public.ubuntu_desktop_provision USING gin (software);
CREATE INDEX idx_ubuntu_desktop_provision_source_metrics ON public.ubuntu_desktop_provision USING gin (source_metrics);
CREATE INDEX idx_ubuntu_release_upgrader_collection_time ON public.ubuntu_release_upgrader USING btree (collection_time);
CREATE INDEX idx_ubuntu_release_upgrader_entry_time_optout ON public.ubuntu_release_upgrader USING btree (entry_time, optout);
CREATE INDEX idx_ubuntu_release_upgrader_hardware ON public.ubuntu_release_upgrader USING gin (hardware);
CREATE INDEX idx_ubuntu_release_upgrader_platform ON public.ubuntu_release_upgrader USING gin (platform);
CREATE INDEX idx_ubuntu_release_upgrader_report_id ON public.ubuntu_release_upgrader USING btree (report_id);
CREATE INDEX idx_ubuntu_release_upgrader_software ON public.ubuntu_release_upgrader USING gin (software);
CREATE INDEX idx_ubuntu_release_upgrader_source_metrics ON public.ubuntu_release_upgrader USING gin (source_metrics);
CREATE INDEX idx_ubuntu_report_distribution_version ON public.ubuntu_report USING btree (distribution, version);
CREATE INDEX idx_ubuntu_report_entry_time_optout ON public.ubuntu_report USING btree (entry_time, optout);
CREATE INDEX idx_ubuntu_report_report ON public.ubuntu_report USING gin (report);
CREATE INDEX idx_ubuntu_report_report_id ON public.ubuntu_report USING btree (report_id);
CREATE INDEX idx_windows_collection_time ON public.windows USING btree (collection_time);
CREATE INDEX idx_windows_entry_time_optout ON public.windows USING btree (entry_time, optout);
CREATE INDEX idx_windows_hardware ON public.windows USING gin (hardware);
CREATE INDEX idx_windows_platform ON public.windows USING gin (platform);
CREATE INDEX idx_windows_report_id ON public.windows USING btree (report_id);
CREATE INDEX idx_windows_software ON public.windows USING gin (software);
CREATE INDEX idx_windows_source_metrics ON public.windows USING gin (source_metrics);
CREATE INDEX idx_wsl_setup_collection_time ON public.wsl_setup USING btree (collection_time);
CREATE INDEX idx_wsl_setup_entry_time_optout ON public.wsl_setup USING btree (entry_time, optout);
CREATE INDEX idx_wsl_setup_hardware ON public.wsl_setup USING gin (hardware);
CREATE INDEX idx_wsl_setup_platform ON public.wsl_setup USING gin (platform);
CREATE INDEX idx_wsl_setup_report_id ON public.wsl_setup USING btree (report_id);
CREATE INDEX idx_wsl_setup_software ON public.wsl_setup USING gin (software);
CREATE INDEX idx_wsl_setup_source_metrics ON public.wsl_setup USING gin (source_metrics);
