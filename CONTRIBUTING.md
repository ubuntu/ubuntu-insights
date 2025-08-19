# Contributing to Ubuntu Insights

A big welcome and thank you for considering contributing to Ubuntu Insights and Ubuntu! It’s people like you that make it a reality for users in our community.

Reading and following these guidelines will help us make the contribution process easy and effective for everyone involved. It also communicates that you agree to respect the time of the developers managing and developing this project. In return, we will reciprocate that respect by addressing your issue, assessing changes, and helping you finalize your pull requests.

These are mostly guidelines, not rules. Use your best judgment, and feel free to propose changes to this document in a pull request.

## Quicklinks

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
    - [Issues](#issues)
    - [Pull Requests](#pull-requests)
- [Contributing to the code](#contributing-to-the-code)
    - [Required dependencies](#required-dependencies)
    - [Building and running the binaries](#building-and-running-the-binaries)
    - [About the test suite](#about-the-test-suite)
      - [Tests with dependencies](#tests-with-dependencies)
    - [Code style](#code-style)
- [Contributor License Agreement](#contributor-license-agreement)
- [Getting Help](#getting-help)

## Code of Conduct

We take our community seriously and hold ourselves and other contributors to high standards of communication. By participating and contributing to this project, you agree to uphold our [Code of Conduct](https://ubuntu.com/community/code-of-conduct).

## Getting Started

Contributions are made to this project via Issues and Pull Requests (PRs). A few general guidelines that cover both:

- To report security vulnerabilities, please use the advisories page of the repository and not a public bug report. Please use [launchpad private bugs](https://bugs.launchpad.net/ubuntu/+source/ubuntu-insights/+filebug) which is monitored by our security team. Alternatively, use the repository's [security page](https://github.com/ubuntu/ubuntu-insights/security). On an Ubuntu machine, it’s best to use `ubuntu-bug ubuntu insights` to collect relevant information.
- Search for existing Issues and PRs on this repository before creating your own.
- We work hard to makes sure issues are handled in a timely manner but, depending on the impact, it could take a while to investigate the root cause. A friendly ping in the comment thread to the submitter or a contributor can help draw attention if your issue is blocking.
- If you've never contributed before, see [this Ubuntu resource post](https://ubuntu.com/community/contribute) for resources and tips on how to get started.

### Issues

Issues should be used to report problems with the software, request a new feature, or to discuss potential changes before a PR is created. When you create a new Issue, a template will be loaded that will guide you through collecting and providing the information we need to investigate.

If you find an Issue that addresses the problem you're having, please add your own reproduction information to the existing issue rather than creating a new one. Adding a [reaction](https://github.blog/2016-03-10-add-reactions-to-pull-requests-issues-and-comments/) can also help be indicating to our maintainers that a particular problem is affecting more than just the reporter.

### Pull Requests

PRs to our project are always welcome and can be a quick way to get your fix or improvement slated for the next release. In general, PRs should:

- Only fix/add the functionality in question **OR** address wide-spread whitespace/style issues, not both.
- Add unit or integration tests for fixed or changed functionality.
- Address a single concern in the least number of changed lines as possible.
- Include documentation in the repo or on our [docs site](https://github.com/ubuntu/Ubuntu-Insights/wiki).
- Be accompanied by a complete Pull Request template (loaded automatically when a PR is created).

For changes that address core functionality or would require breaking changes (e.g. a major release), it's best to open an Issue to discuss your proposal first. This is not required but can save time creating and reviewing changes.

In general, we follow the ["fork-and-pull" Git workflow](https://github.com/susam/gitpr)

1. Fork the repository to your own GitHub account
1. Clone the project to your machine
1. Create a branch locally with a succinct but descriptive name
1. Commit changes to the branch
1. Following any formatting and testing guidelines specific to this repo
1. Push changes to your fork
1. Open a PR in our repository and follow the PR template so that we can efficiently review the changes.

> PRs will trigger unit and integration tests with and without race detection, linting and formatting validations, static and security checks, freshness of generated files verification. All the tests must pass before merging in main branch.

Once merged to the main branch, `po` files and any documentation change will be automatically updated. Those are thus not necessary in the pull request itself to minimize diff review.

## Contributing to the code

### Required dependencies

This project has several build dependencies. On Ubuntu, you can install these dependencies from the `insights/` directory using the apt command as follows:

```bash
sudo apt update
sudo apt build-dep .
sudo apt install devscripts
```

On other operating systems, you will need [Go](https://go.dev/) at a minimum.

### Building and running the binaries

This repository is set up in a mono-repo structure, and consists of multiple modules and binaries.

- `insights/`: The client
  - `ubuntu-insights`: The CLI
  - `libinsights.so`/`libinsights.dll`: C shared/dynamic library
- `server/`: Server services
  - `web-service`: Server HTTP server service
  - `ingest-service`: Server database ingest service
- `common/`: A helper shared Go package dependency

The project's client components can be built as a Debian package. This process will compile all the client-side binaries, run the test suite and produce the Debian packages.

Alternatively, for development purposes or for the server services, each binary can be built manually and separately.

#### Building the client Debian packages from source

Building the Debian packages from source is the most straightforward and standard method for compiling the binaries and running the test suite. To do this, run the following command from `insights/` folder of the source tree:

```shell
debuild
```

The Debian packages are available in the parent directory.

#### Building the CLIs only

To build a CLI only, either the client, or a server service, run the appropriate command from the top of the source tree.
The resulting binary will be found in the current directory, and can be run directly without needing to be installed.

##### Client

```shell
go build -o ubuntu-insights ./insights/cmd/insights
```

##### Web Service

```shell
go build ./server/cmd/web-service
```

##### Ingest Service

```shell
go build ./server/cmd/ingest-service
```

#### Building C bindings only

To build the C bindings only, run the following command from the top of the source tree:

```shell
go generate ./insights/C/...
```

The resulting binaries and associated C header files will be in the `insights/generated/` folder.

### About the test suite

The project includes a comprehensive test suite made of unit and integration tests. All the tests must pass before the review is considered. If you have troubles with the test suite, feel free to mention it on your PR description.

To run all tests for a given module, from the module's root folder (i.e., `insights/`, `server/`, or `common/`), run: `go test ./...` (add the `-race` flag for race detection).

The test suite must pass before merging the PR to our main branch. Any new feature, change or fix must be covered by corresponding tests.

#### Tests with dependencies

Some server integration tests use the [Testcontainers Go package](https://golang.testcontainers.org/) which requires a [Docker-API compatible container runtime](https://golang.testcontainers.org/system_requirements/docker/).

These special test dependencies are not included in the project dependencies and must be installed manually.

### Code style

This project follow the Go code-style. The client C bindings roughly follow the GNU code-style. For more informative information about the code style in use, please check:

- For Go: <https://google.github.io/styleguide/go/>
- For C: <https://www.gnu.org/prep/standards/html_node/Writing-C.html>

## Contributor License Agreement

It is required to sign the [Contributor License Agreement](https://ubuntu.com/legal/contributors) in order to contribute to this project.

An automated test is executed on PRs to check if it has been accepted.

This project is covered by [GPL-3.0](LICENSE).

## Getting Help

Join us in the [Ubuntu Community](https://discourse.ubuntu.com/c/desktop/8) and post your question there with a descriptive tag.
