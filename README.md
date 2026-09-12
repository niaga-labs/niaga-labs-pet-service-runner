# service-runner

Runner profile and availability management service with geospatial query capabilities.

## Description

This service manages runner profiles, online/offline availability sessions, real-time GPS location updates, and crate specifications. It provides geospatial queries using PostGIS to find nearby available runners for booking requests.

## Features

- Runner profile creation and management
- Online/offline session tracking
- Real-time GPS location updates
- Crate specifications for pet transport
- PostGIS-powered nearby runner search
- Quality policy enforcement
- Kafka event publishing for runner state changes

## API Endpoints

| Method | Endpoint                      | Access   | Description                    |
|--------|-------------------------------|----------|--------------------------------|
| POST   | /api/v1/runners               | Runner   | Create runner profile          |
| GET    | /api/v1/runners/me            | Runner   | Get current runner profile     |
| POST   | /api/v1/runners/me/online     | Runner   | Set runner online              |
| POST   | /api/v1/runners/me/offline    | Runner   | Set runner offline             |
| POST   | /api/v1/runners/me/location   | Runner   | Update GPS location            |
| POST   | /api/v1/runners/me/crates     | Runner   | Update crate specifications    |
| GET    | /api/v1/runners/nearby        | Internal | Find nearby available runners  |

## Kafka Events Published

- **runner.online**: Runner becomes available
- **runner.offline**: Runner goes offline
- **runner.location_update**: GPS location updated

## Configuration

The service requires the following environment variables:

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=runner_db
SERVICE_PORT=8003
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC_PREFIX=kilat-pet-runner
```

## Tech Stack

- **Language**: Go 1.24
- **Web Framework**: Gin
- **ORM**: GORM
- **Database**: PostgreSQL with PostGIS extension
- **Message Queue**: Kafka (shopify/sarama)
- **Geospatial**: PostGIS for location queries

## Running the Service

```bash
# Install dependencies
go mod download

# Enable PostGIS extension
psql -d runner_db -c "CREATE EXTENSION IF NOT EXISTS postgis;"

# Point at the shared dev-infra stack (cd ~/Documents/dev-infra; ./dev.ps1 up kilat)
export DB_HOST=localhost DB_PORT=5432 DB_USER=kilat DB_PASSWORD=kilat_secret
export DB_NAME=kilat_runner DB_SSL_MODE=disable

# Apply the SQL migrations -- run from the repository root, the migration
# source is resolved relative to the working directory
go run ./cmd/migrate

# Start the service
go run ./cmd/server
```

### Migrations

`migrations/` is the single source of truth for the schema, in every environment including
development. `cmd/migrate` applies the files and exits; `cmd/server` applies them at startup
too, so a fresh boot is self-sufficient and you do not have to remember a separate step.

There is deliberately no GORM `AutoMigrate` path. Until KPD-59 the server auto-migrated in
development and ran the SQL migrations everywhere else, and `pet_shops` had no SQL migration --
so it existed only on a developer laptop. One path for all environments is what stops that
class of drift returning.

Sample pet shops are still seeded in development only, by `repository.SeedPetShops`.

The service will start on port 8003.

## Database Schema

- **runners**: Runner profiles with user reference
- **sessions**: Online/offline availability sessions
- **crates**: Transport crate specifications
- **locations**: GPS coordinates with PostGIS geometry type
