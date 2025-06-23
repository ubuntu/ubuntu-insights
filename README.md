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

This is the code repository for **Ubuntu Insights**, a transparent, user-friendly, open, platform-agnostic, and cross-application solution for reporting hardware information and other collected metrics.

Ubuntu Insights is designed to show you exactly what is being sent, and allow you to acknowledge and control your own data. The code in this repository is designed to mainly be invoked by a controlling application, but a command-line tool is also provided.

Ubuntu Insights is designed to be a full replacement for [Ubuntu Report](https://github.com/ubuntu/ubuntu-report) and its server components are backwards compatible.

## Components

Ubuntu Insights is divided into two components:

- The client for local actions which consists of a command-line interface, as well as Go and C APIs. It is also packaged as a deb package. See [insights](insights)
- The server services which is what we use to aggregate reports. See [server](server)

## Example Insights Reports

### Ubuntu Desktop

```json
{
  "insightsVersion": "Dev",
  "collectionTime": 1748013676,
  "systemInfo": {
    "hardware": {
      "product": {
        "family": "My Product Family",
        "name": "My Product Name",
        "vendor": "My Product Vendor"
      },
      "cpu": {
        "name": "9 1200SX",
        "vendor": "Authentic",
        "architecture": "x86_64",
        "cpus": 16,
        "sockets": 1,
        "coresPerSocket": 8,
        "threadsPerCore": 2
      },
      "gpus": [
        {
          "device": "0x0294",
          "vendor": "0x10df",
          "driver": "gpu"
        },
        {
          "device": "0x03ec",
          "vendor": "0x1003",
          "driver": "gpu"
        }
      ],
      "memory": {
        "size": 23247
      },
      "disks": [
        {
          "size": 1887436,
          "type": "disk",
          "children": [
            {
              "size": 750,
              "type": "part"
            },
            {
              "size": 260,
              "type": "part"
            },
            {
              "size": 16,
              "type": "part"
            },
            {
              "size": 1887436,
              "type": "part"
            },
            {
              "size": 869,
              "type": "part"
            },
            {
              "size": 54988,
              "type": "part"
            }
          ]
        }
      ],
      "screens": [
        {
          "physicalResolution": "2560x1440",
          "size": "600mm x 340mm",
          "refreshRate": "143.85"
        },
        {
          "physicalResolution": "2560x1600",
          "size": "300mm x 190mm",
          "refreshRate": "120.00"
        }
      ]
    },
    "software": {
      "os": {
        "family": "linux",
        "distribution": "Ubuntu",
        "version": "25.04"
      },
      "timezone": "EDT",
      "language": "en_US",
      "bios": {
        "vendor": "Bios Vendor",
        "version": "Bios Version"
      }
    },
    "platform": {
      "desktop": {
        "desktopEnvironment": "ubuntu:GNOME",
        "sessionName": "ubuntu",
        "sessionType": "wayland"
      },
      "proAttached": true
    }
  }
}
```

### WSL

```json
{
  "insightsVersion": "Dev",
  "collectionTime": 1748012492,
  "systemInfo": {
    "hardware": {
      "cpu": {
        "name": "AM Cirus 1200XK Processor",
        "vendor": "Authentic",
        "architecture": "x86_64",
        "cpus": 256,
        "sockets": 1,
        "coresPerSocket": 128,
        "threadsPerCore": 2
      },
      "memory": {
        "size": 15904
      },
      "disks": [
        {
          "size": 388,
          "type": "disk"
        },
        {
          "size": 185,
          "type": "disk"
        },
        {
          "size": 4096,
          "type": "disk"
        },
        {
          "size": 1048576,
          "type": "disk"
        },
        {
          "size": 1048576,
          "type": "disk"
        }
      ]
    },
    "software": {
      "os": {
        "family": "linux",
        "distribution": "Ubuntu",
        "version": "25.04"
      },
      "timezone": "EDT",
      "language": "en_GB"
    },
    "platform": {
      "wsl": {
        "subsystemVersion": 2,
        "systemd": "used",
        "interop": "enabled",
        "version": "2.5.7.0",
        "kernelVersion": "6.6.87.1-microsoft-standard-WSL2"
      },
      "proAttached": true
    }
  }
}
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
- For suggestions and constructive feedback, open a feature request issue using the proposed template.

## Get in touch

We're friendly! We have a community forum at [https://discourse.ubuntu.com](https://discourse.ubuntu.com) where we discuss feature plans, development news, issues, updates and troubleshooting.

For news and updates, follow the [Ubuntu Twitter account](https://twitter.com/ubuntu) and on [Facebook](https://www.facebook.com/ubuntu).
