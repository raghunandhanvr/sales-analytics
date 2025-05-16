# Sales Analytics API

High-throughput CSV → MySQL ingestion + analytics in Go.

![ER Diagram](docs/db/er.png)

## What It Does

This API handles two main tasks:
- **Data Ingestion**: Imports CSV sales data into a MySQL database
- **Analytics**: Provides endpoints to query and analyze the imported data

## API Documentation

The full API is documented on [SwaggerHub](https://app.swaggerhub.com/apis/freightify-65d/Lumel/1.0).

## Technical Implementation
* **Dynamic Programming Approach**
  * Two-pass algorithm for efficient entity processing
  * Memoization tables for unique customers and products
  * Dependency-aware processing order (prerequisites first)
  * Optimized batch sizes based on benchmarks

* **Concurrent File Processing**
  * Multiple files handled via separate job contexts
  * Atomic counters for lock-free row tallying
  * Optimized connection pooling (20 max, 10 idle)
  * Worker goroutines with controlled concurrency

* **Duplicate Management**
  * Hash-based deduplication via map data structures
  * Unique constraints in database schema
  * REPLACE INTO for handling duplicate entities
  * Optimistic locking for concurrent operations

## Setup

### Prerequisites
| Tool | Version | Purpose |
|------|---------|---------|
| Go | ≥ 1.22 | compile |
| MySQL | 8.x | data |
| (opt) Docker | latest | for swagger-ui |

### Setting up the Database

- You can refer the (docs/db/migrations.sql) to create the tables

### Starting the server

```bash
git clone https://github.com/raghunandhanvr/sales-analytics.git
cd sales-analytics

# copy & edit config
cp config/config.yaml.example config/config.yaml
$EDITOR config/config.yaml   # add your MySQL creds, CSV path

go mod tidy
go install github.com/google/wire/cmd/wire@latest
wire ./internal/di           # generate DI code

## Running the API

```bash
go run ./cmd/server  # Starts on port 8080 by default
```

### Example: Uploading CSV Data

```bash
curl -F file=@sample_data.csv http://localhost:8080/api/v1/ingestion/upload
```

## Configuration

Example `config/config.yaml`:

```yaml
app:
  port: 8080
  mode: debug        # debug | release
db:
  user: YOUR_USER
  password: YOUR_PASS
  host: 127.0.0.1
  port: "3306"
  name: sales_db
csv:
  path: /path/to/sample_data.csv
cron:
  spec: "0 0 * * *"  # daily at midnight
```

## Project Structure
```
cmd/server         → main entry, Wire injector
config/            → Viper YAML
docs/
  swagger/         → OpenAPI 3.0 spec
  db/              → ER diagram
internal/
  di/              → providers + wire_gen.go
  handler/         → Gin controllers
  service/         → business logic (ingestion / analytics)
  repository/      → SQL access layer
  models/          → pure structs
  utils/           → response helpers
  errors/          → central API errors
pkg/orm/           → tiny helper (exec/query wrappers)
sample_data.csv    → demo file
```
