# Ubuntu Insights Client

[reference-documentation-image]: https://pkg.go.dev/badge/github.com/ubuntu/ubuntu-insights/insights.svg
[reference-documentation-url]: https://pkg.go.dev/github.com/ubuntu/ubuntu-insights/insights

[![Go Reference][reference-documentation-image]][reference-documentation-url]

The Ubuntu Insights Client is the component responsible for handling all local Ubuntu Insights actions. This component includes a command-line interface, a Go and C API, as well as the source for the `ubuntu-insights` deb package.

## About

Ubuntu Insights caches all collected data locally, and will only attempt to upload insights reports following a minimum period (1 week by default). It checks for consent both at the time of collection and at the time of upload. If consent was not given at either of those two points, then the information is not sent to the server. Once a report has been uploaded, it is removed from the `local` cache folder, and the data that was sent is written to the `uploaded` folder.

By default, Ubuntu Insights will only collect once per collection period.

A system-wide opt-out can be configured by administrators to override all per-user and per-source consent settings. When active, all collection and upload operations behave as if consent is denied.

To execute the interactive command-line interface manually, use `ubuntu-insights`.

### Example Insights Reports

See [here](/#example-insights-reports)

## Default Paths

### Linux

Consent Files: `~/.config/ubuntu-insights`

Reports Cache: `~/.cache/ubuntu-insights`

System Configuration: `/etc/ubuntu-insights`

## Command-Line Interface Usage

### ubuntu-insights

`ubuntu-insights [command]`

#### Options

```none
Available Commands:
  collect         Collect system information
  completion      Generate the autocompletion script for the specified shell
  consent         Manage or get user consent state
  help            Help about any command
  system-opt-out  Manage or get the system-wide opt-out state
  upload          Upload metrics to the Ubuntu Insights server

Flags:
      --config string              use a specific configuration file
      --consent-dir string         the base directory of the consent state files
  -h, --help                       help for ubuntu-insights
      --insights-dir string        the base directory of the insights report cache
  -q, --quiet                      suppress all output except errors
      --system-config-dir string   the directory of the system-wide configuration file
  -v, --verbose count              issue INFO (-v), DEBUG (-vv)
      --version                    version for ubuntu-insights
```

### ubuntu-insights consent

Manage or get user consent state for data collection and upload

`ubuntu-insights consent [sources](optional arguments) [flags]`

#### Options

```none
Flags:
  -h, --help           help for consent
  -s, --state string   the consent state to set (true or false)

Global Flags:
      --config string        use a specific configuration file
      --consent-dir string   the base directory of the consent state files
  -v, --verbose count        issue INFO (-v), DEBUG (-vv)
```

#### Examples

To get the platform consent state (linux/windows/darwin depending on your platform) using the default consent directory:

```console
foo@bar:~$ ubuntu-insights consent
linux: false
```

To set the consent state for the source `wsl_setup`:

```console
foo@bar:~$ ubuntu-insights consent wsl_setup -s true
wsl_setup: true
```

### ubuntu-insights system-opt-out

Manage or get the system-wide opt-out state for data collection and upload.

When the system opt-out is active, all collection and upload operations behave as if consent is denied, regardless of per-user or per-source consent settings.

Setting the system opt-out state typically requires administrative privileges to write to the system configuration directory.

`ubuntu-insights system-opt-out [flags]`

#### Options

```none
Flags:
  -h, --help           help for system-opt-out
  -s, --state string   the system opt-out state to set (true or false)

Global Flags:
      --config string              use a specific configuration file
      --system-config-dir string   the directory of the system-wide configuration file
  -q, --quiet                      suppress all output except errors
  -v, --verbose count              issue INFO (-v), DEBUG (-vv)
```

#### Examples

To get the current system-wide opt-out state:

```console
foo@bar:~$ ubuntu-insights system-opt-out
false
```

To enable the system-wide opt-out (may require administrative privileges):

```console
foo@bar:~$ sudo ubuntu-insights system-opt-out -s true
true
```

To disable the system-wide opt-out:

```console
foo@bar:~$ sudo ubuntu-insights system-opt-out -s false
false
```

### ubuntu-insights collect

Collect system information and metrics and store it locally.

If source is not provided, then the source is assumed to be the currently detected platform. Additionally, there should be no source-metrics-path provided.
If source is provided, then the source-metrics-path should be provided as well.

`ubuntu-insights collect [source] [source-metrics-path](required if source provided) [flags]`

#### Options

```none
Flags:
  -d, --dry-run       perform a dry-run where a report is collected, but not written to disk
  -f, --force         force a collection, override the report if there are any conflicts (consent is still respected)
  -h, --help          help for collect
  -p, --period uint   the minimum period between 2 collection periods for validation purposes in seconds (default 1)

Global Flags:
      --config string         use a specific configuration file
      --consent-dir string    the base directory of the consent state files
      --insights-dir string   the base directory of the insights report cache
  -v, --verbose count         issue INFO (-v), DEBUG (-vv)
```

### ubuntu-insights upload

Upload metrics to the Ubuntu Insights server.

If no sources are provided, all detected sources at the configured reports directory will be uploaded.
If consent is not given for a source, an opt-out notification will be sent regardless of the locally cached insights report's contents.

`ubuntu-insights upload [sources](optional arguments) [flags]`

#### Options

```none
Flags:
  -d, --dry-run        go through the motions of doing an upload, but do not communicate with the server, send the payload, or modify local files
  -f, --force          force an upload, ignoring min age and clashes between the collected file and a file in the uploaded folder, replacing the clashing uploaded report if it exists (doesn't ignore consent)
  -h, --help           help for upload
      --min-age uint   the minimum age (in seconds) of a report before the uploader will attempt to upload it (default 604800)
  -r, --retry          enable a limited number of retries for failed uploads

Global Flags:
      --config string         use a specific configuration file
      --consent-dir string    the base directory of the consent state files
      --insights-dir string   the base directory of the insights report cache
  -v, --verbose count         issue INFO (-v), DEBUG (-vv)
```

### Hidden commands

These commands are hidden from help, and should primarily be used by the system or for debugging.

#### ubuntu-insights completion

Generate the autocompletion script for ubuntu-insights for the specified shell.

`ubuntu-insights completion [shell]`
