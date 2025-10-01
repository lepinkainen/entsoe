# CLAUDE.md

Refer to llm-shared/project_tech_stack.md for library and technology choices.

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application that fetches electricity price data from the ENTSOE (European Network of Transmission System Operators for Electricity) API and stores it in Redis Time Series. The application is designed to run as a scheduled job to collect daily electricity pricing data for Finland.

## Core Architecture

The application consists of two main Go files:

- `main.go` - Main application logic with data fetching and Redis storage
- `entsoe.go` - XML data structures for parsing ENTSOE API responses

Key components:

- **Configuration**: Uses Viper for YAML config management (`config.yml`)
- **Data Storage**: Redis Time Series with duplicate policy "LAST"
- **API Integration**: ENTSOE Transparency Platform REST API
- **Data Processing**: XML unmarshaling of market documents with hourly price points

## Development Commands

The project uses Task (Taskfile.yml) for build automation:

```bash
# Build the application (runs tests, vet, and lint first)
task build

# Build for Linux deployment
task build-linux

# Run tests
task test

# Run tests for CI (includes coverage)
task test-ci

# Run a single test
go test -v -run TestName ./...

# Run Go linter
task lint

# Run Go vet
task vet

# Upgrade all dependencies
task upgrade-deps

# Deploy to server
task publish
```

Standard Go commands:

```bash
# Run the application
go run .

# Run with debug mode (prints data without storing to Redis)
go run . -debug

# Run specific test file
go test -v entsoe_test.go entsoe.go

# Format code with goimports (preferred over gofmt)
goimports -w .
```

## Configuration

The application expects a `config.yml` file with Redis connection details and ENTSOE API credentials. It searches for config in:

- `$HOME/.config/entsoe_redis/config.yml`
- `$HOME/.entsoe_redis/config.yml`
- `./config.yml`

Required config structure:

```yaml
redis:
  address: "host:port"
  username: "username"
  password: "password"
  db: 0
  dbname: "timeseries_key_name"

nordpool:
  apikey: "api_key"
  in_domain: "domain_code"
  out_domain: "domain_code"
```

## Key Implementation Details

### Data Flow

1. **Fetch**: HTTP GET to ENTSOE API with date range and domain parameters
2. **Parse**: XML unmarshaling into `PublicationMarketDocument` or `AcknowledgementMarketDocument` (for errors)
3. **Transform**: Convert hourly price points from EUR/MWh to c/kWh, calculate timestamps
4. **Store**: Batch insert into Redis Time Series using pipelining

### Technical Details

- **Time Series Storage**: Creates Redis TS with labels `type:price, country:fi`
- **Data Pipeline**: Uses Redis pipelining for efficient batch inserts
- **Retry Logic**: HTTP client retries up to 3 times with exponential backoff (2s, 4s, 6s)
- **Error Handling**: Handles both successful data responses and ENTSOE error documents (`AcknowledgementMarketDocument`)
- **Time Processing**:
  - Parses ISO 8601 duration format (PT1H, PT15M, etc.) via `isoResolutionToDuration()`
  - Converts ENTSOE timestamps to Unix milliseconds for Redis
  - Handles hourly, minute, and second resolutions
- **Price Conversion**: Divides API prices by 10 to get c/kWh from EUR/MWh
- **Debug Mode**: `--debug` flag prints parsed data without storing to Redis
- **Version Embedding**: Build process embeds git commit hash via `-ldflags="-X main.Version={{.GIT_COMMIT}}"`

### Testing Strategy

- Tests use `//go:build !ci` tag to allow skipping in CI environments when needed
- Tests cover XML unmarshaling, data structures, and ISO resolution parsing
- Run normal tests with `task test`, CI tests with `task test-ci`

### Missing Configurations (Not Critical)

- No `.golangci.yml` - uses default golangci-lint settings
- No GitHub Actions CI - application is deployed manually via `task publish`
