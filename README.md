# Welcome to Ubuntu Insights

[actions-image]: https://github.com/ubuntu/ubuntu-insights/actions/workflows/qa.yaml/badge.svg?branch=main
[actions-url]: https://github.com/ubuntu/ubuntu-insights/actions?query=branch%3Amain+event%3Apush
[license-image]: https://img.shields.io/badge/License-GPL3.0-blue.svg
[codecov-image]: https://codecov.io/gh/ubuntu/ubuntu-insights/branch/main/graph/badge.svg
[codecov-url]: https://codecov.io/gh/ubuntu/ubuntu-insights
[goreport-image]: https://goreportcard.com/badge/github.com/ubuntu/ubuntu-insights
[goreport-url]: https://goreportcard.com/report/github.com/ubuntu/ubuntu-insights

[![Code quality][actions-image]][actions-url]
[![License][license-image]](LICENSE)
[![Code coverage][codecov-image]][codecov-url]
[![Go Report Card][goreport-image]][goreport-url]

This is the code repository for **Ubuntu Insights**, a user transparent, open, platform-agnostic and cross application solution for reporting hardware information and other collected metrics.

Ubuntu Insight is designed to show you exactly what is being sent, and allow you to acknowledge and control your own data. The code in this repository is designed to mainly be invoked by a controlling application, but a command line tool is also provided.

This is designed to be a full replacement for Ubuntu Report.

## About

## Command Line Usage

### ubuntu-insights

`ubuntu-insights [command]`

#### Options

```
Available Commands:
  collect     Collect system information
  completion  Generate the autocompletion script for the specified shell
  consent     Manage or get user consent state
  help        Help about any command
  upload      Upload metrics to the Ubuntu Insights server

Flags:
      --config string         use a specific configuration file
      --consent-dir string    the base directory of the consent state files
  -h, --help                  help for ubuntu-insights
      --insights-dir string   the base directory of the insights report cache
  -v, --verbose count         issue INFO (-v), DEBUG (-vv)
      --version               version for ubuntu-insights
```

### ubuntu-insights consent

Manage or get user consent state for data collection and upload

`ubuntu-insights consent [sources](optional arguments) [flags]`

#### Options

```
Flags:
  -h, --help           help for consent
  -s, --state string   the consent state to set (true or false)

Global Flags:
      --config string        use a specific configuration file
      --consent-dir string   the base directory of the consent state files
  -v, --verbose count        issue INFO (-v), DEBUG (-vv)
```

#### Examples

To get the global consent state using the default consent directory:

```console
foo@bar:~$ ubuntu-insights consent
Global: false
```

To set the consent state for the source `linux`:

```console
foo@bar:~$ ubuntu-insights consent linux -s true
linux: true
```

### ubuntu-insights collect

Collect system information and metrics and store it locally.

If source is not provided, then the source is assumed to be the currently detected platform. Additionally, there should be no source-metrics-path provided.
If source is provided, then the source-metrics-path should be provided as well.

`ubuntu-insights collect [source] [source-metrics-path](required if source provided) [flags]`

#### Options

```
Flags:
  -d, --dry-run       perform a dry-run where a report is collected, but not written to disk
  -f, --force         force a collection, override the report if there are any conflicts (doesn't ignore consent)
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

```
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

## Example Insights Reports

### Ubuntu Desktop
```json
```

### WSL

```json
```

### Data being sent if consent is false

```json
{
    "OptOut": true
}
```

## Get involved

This is an [open source](LICENSE) project, and we warmly welcome community contributions, suggestions, and constructive feedback. If you're interested in contributing, please take a look at our [Contribution guidelines](CONTRIBUTING.md) first.

- To report an issue, please file a bug report against our repository, using a bug template.
- For suggestions and constructive feedback, report a feature request bug report, using the proposed template.

## Get in touch

We're friendly! We have a community forum at [https://discourse.ubuntu.com](https://discourse.ubuntu.com) where we discuss feature plans, development news, issues, updates and troubleshooting.

For news and updates, follow the [Ubuntu Twitter account](https://twitter.com/ubuntu) and on [Facebook](https://www.facebook.com/ubuntu).
