# Ubuntu Insights Server Services

The Ubuntu Insights Services are the components for aggregating and processing reports. They do not record any personally identifying information such as IPs.

There are two server services, a web exposed web service which handles incoming HTTP requests as well as an ingest service which does simple validations before inserting reports into a database.

Neither of these services are meant for local use.

## Usage

### Web Service

```shell
  ubuntu-insights-web-service [flags]
  ubuntu-insights-web-service [command]
```

#### Options

```shell
Available Commands:
  help        Help about any command
  version     Returns the running version of ubuntu-insights-web-service and exits

Flags:
      --config string              use a specific configuration file
      --daemon-config string       path to the configuration file
  -h, --help                       help for ubuntu-insights-web-service
      --listen-host string         host to listen on
      --listen-port int            port to listen on (default 8080)
      --max-header-bytes int       maximum header bytes for HTTP server (default 8192)
      --max-upload-bytes int       maximum upload bytes for HTTP server (default 131072)
      --metrics-host string        host for metrics endpoint
      --metrics-port int           port for metrics endpoint (default 2112)
      --read-timeout duration      read timeout for HTTP server (default 5s)
      --reports-dir string         directory to store reports in (default "/var/lib/ubuntu-insights-services/reports")
      --request-timeout duration   request timeout for HTTP server (default 3s)
  -v, --verbose count              issue INFO (-v), DEBUG (-vv)
      --write-timeout duration     write timeout for HTTP server (default 10s)
```

### Ingest Service

```shell
  ubuntu-insights-ingest-service [flags]
  ubuntu-insights-ingest-service [command]
```

#### Options

```shell
Available Commands:
  help        Help about any command
  migrate     Run migration scripts
  version     Returns the running version of ubuntu-insights-ingest-service and exits

Flags:
      --config string          use a specific configuration file
  -c, --daemon-config string   path to the configuration file
      --db-host string         database host
  -n, --db-name string         database name
  -P, --db-password string     database password
  -p, --db-port int            database port (default 5432)
  -s, --db-sslmode string      database SSL mode
  -u, --db-user string         database user
  -h, --help                   help for ubuntu-insights-ingest-service
      --reports-dir string     base directory to read reports from (default "/var/lib/ubuntu-insights-services/reports")
  -v, --verbose count          issue INFO (-v), DEBUG (-vv)
```

### The Daemon Config

The daemon config, which can be passed to either the web service or the ingest service or shared with both, is a required JSON file which consists of an allow list configuration. This file is watched by the service in a manner such that changes to it will be applied without requiring the service to restart.

See [this file](./examples/daemon-config.json) for an example of how it should be formatted.

An application must be included in this list for the web service to accept HTTP requests from a given application endpoint, and for the ingest service to process reports for that application.

Applications or items meant to be treated as legacy reports from Ubuntu Report should be added with the format: `ubuntu-report/<distribution>/desktop/<version>`.

#### Reserved Names

The following applications and items are reserved and cannot be used within the allow list

- `ubuntu_report`
- `schema_migrations`
