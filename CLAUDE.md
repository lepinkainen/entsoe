# CLAUDE.md

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
# Build the application
task build

# Build for Linux deployment  
task build_linux

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

# Test (if tests exist)
go test ./...

# Format code
go fmt ./...
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

- **Time Series Storage**: Creates Redis TS with labels `type:price, country:fi`
- **Data Pipeline**: Uses Redis pipelining for efficient batch inserts
- **Error Handling**: Handles both successful data responses and ENTSOE error documents
- **Time Processing**: Converts ENTSOE timestamps to Unix milliseconds for Redis
- **Price Conversion**: Divides API prices by 10 to get c/kWh from EUR/MWh