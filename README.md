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

## Performance Metrics

The table below shows performance benchmarks for various dataset sizes:

| Dataset Size | Rows   | Processing Time | Rows/Second | Memory Usage | CPU Usage |
|--------------|--------|----------------|-------------|--------------|-----------|
| Small        | 1,000  | 1.2 seconds    | 833/sec     | 48 MB        | 28%       |
| Medium       | 10,000 | 7.5 seconds    | 1,333/sec   | 124 MB       | 64%       |
| Large        | 50,000 | 32.8 seconds   | 1,524/sec   | 256 MB       | 78%       |
| X-Large      | 100,000| 58.3 seconds   | 1,715/sec   | 512 MB       | 85%       |
| Massive      | 500,000| 270.1 seconds  | 1,851/sec   | 1.2 GB       | 92%       |

```json
{
  "level": "info",
  "ts": "2023-07-12T15:23:41.732Z",
  "caller": "ingestion/service.go:285",
  "msg": "ingestion_completed",
  "job_id": "b7c9e836-4e87-42f1-9d20-0a94836f76c2",
  "rows": 100000,
  "customers": 12485,
  "products": 5729,
  "orders": 21348,
  "items": 100000,
  "duration": 58.3,
  "rows_per_sec": 1715.26,
  "parsing_time": 14.2,
  "db_time": 38.6,
  "db_time_percent": 66.21
}
```

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
## Is this ready for Production?

No.

- To set this project to production grade, consider having a SQS or any queue mechanism to poll the files
- Have notification  mechanism on failure
- Have Authentication/Authorization
- Have Audit Logs