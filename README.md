# entsoe_redis

Fetches electricity day-ahead prices from the [ENTSOE Transparency Platform](https://transparency.entsoe.eu/) API and stores them in Redis Time Series. Designed to run as a daily cron job.

Prices are converted from EUR/MWh to c/kWh. Currently configured for Finland.

## Configuration

Create `config.yml` in one of:
- `$HOME/.config/entsoe_redis/`
- `$HOME/.entsoe_redis/`
- `./`

```yaml
redis:
  address: "host:port"
  username: "username"
  password: "password"
  db: 0
  dbname: "timeseries_key_name"

nordpool:
  apikey: "your_entsoe_api_key"
  in_domain: "10YFI-1--------U"
  out_domain: "10YFI-1--------U"
```

Get an API key from the [ENTSOE Transparency Platform](https://transparency.entsoe.eu/).

## Usage

```
entsoe_redis           # fetch today's prices and store in Redis
entsoe_redis -debug    # print prices to stdout without storing
```

## Building

Requires [Task](https://taskfile.dev/).

```
task build             # test, vet, lint, then build
task build-linux       # cross-compile for linux/amd64
task test              # run tests
task lint              # run golangci-lint
```
