# PDA Monitor Service

Lightweight Go-based telemetry monitor that ingests third-party telemetry data, exposes it via a REST API, sends Telegram notifications, and produces spreadsheet-style reports. Designed for Linux-first deployment with **user-level systemd supervision** and **`.env`-based runtime configuration**.

---

## Core Capabilities

* Telemetry data ingestion from external API
* REST API for downstream consumers
* JWT-based authentication with protected admin routes
* Formula and alert-level management endpoints
* Telegram bot notifications
* Daily and weekly XLSX reporting output
* MariaDB/MySQL backend
* Fully `.env`-driven configuration
* User-level `systemd` service (no root required)

---

## Tech Stack

* **Language:** Go
* **Database:** MariaDB / MySQL
* **API Style:** REST over Chi
* **Messaging:** Telegram Bot API
* **Process Manager:** systemd (user-level)

---

## Project Layout

```text
.
├── api_docs.md
├── bin/
│   ├── pda-dbinit
│   └── pda-monitor
├── cmd/
│   ├── dbinit/
│   │   └── main.go
│   └── server/
│       └── main.go
├── go.mod
├── go.sum
├── internal/
│   ├── auth/
│   │   └── jwt.go
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── schema.go
│   ├── domain/
│   │   ├── alert_level.go
│   │   ├── formula.go
│   │   ├── reading.go
│   │   ├── station.go
│   │   └── user.go
│   ├── handler/
│   │   ├── alert_handler.go
│   │   ├── api_handler.go
│   │   ├── auth_handler.go
│   │   ├── export_handler.go
│   │   ├── formula_handler.go
│   │   ├── reading_handler.go
│   │   └── station_handler.go
│   ├── httpx/
│   │   └── httpx.go
│   ├── logger/
│   │   └── logger.go
│   ├── middleware/
│   │   └── auth.go
│   ├── notification/
│   │   └── telegram.go
│   ├── parser/
│   │   └── telemetry_parser.go
│   ├── report/
│   │   └── excel.go
│   ├── repository/
│   │   ├── interfaces.go
│   │   └── mysql/
│   │       ├── alert_level_repo.go
│   │       ├── formula_repo.go
│   │       ├── reading_repo.go
│   │       ├── station_repo.go
│   │       └── user_repo.go
│   ├── scheduler/
│   │   └── scheduler.go
│   ├── service/
│   │   ├── debit_calculator.go
│   │   ├── reading_service.go
│   │   ├── report_service.go
│   │   └── telemetry_service.go
│   ├── timeutil/
│   │   └── timeutil.go
│   └── util/
│       └── station.go
├── Makefile
├── planning/
│   └── backend-performance-sql-revision-plan.md
└── scripts/
    ├── apply-system-dbinit.sh
    ├── redeploy-system-with-dbinit.sh
    └── redeploy.sh
```

---

## API Overview

Protected API routes are mounted under `/api` and require JWT authentication, except for `POST /api/auth/login`.

Primary route groups:

* `/api/readings/*` for current, latest, station-specific, and historical readings
* `/api/pda/*` for realtime and historical telemetry with debit calculation
* `/api/stations/*` for station listing and sync
* `/api/formulas/*` for rating-curve retrieval and admin-managed updates
* `/api/alert-levels/*` for alert threshold retrieval and admin-managed updates
* `/api/export/*` for daily and weekly XLSX exports

Admin-only write routes are enforced through the JWT role middleware.

---

## Runtime Environment Variables

All runtime configuration is supplied via a `.env` file.

Required variables for the API server:

```env
PORT=8080

DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=secret
DB_NAME=pda_monitor

TELEMETRY_BASE_URL=https://example.com
TELEMETRY_IDBBWS=xxxx
TELEMETRY_TIMEOUT=30

TELEGRAM_BOT_TOKEN=xxxx
TELEGRAM_CHAT_IDS=xxxx
TELEGRAM_CHANNELS=xxxx

JWT_EXPIRY_HOURS=24

# Logging
LOG_LEVEL=INFO          # DEBUG, INFO, WARN, ERROR
LOG_FILE=/path/to/log/pda-monitor/app.log  # Leave empty for no file logging
LOG_CONSOLE=true        # Output to console/stdout
```

### Logging Behavior

* `LOG_CONSOLE=true` → logs are always written to stdout (captured by systemd journal).
* `LOG_FILE` → optional secondary file log. If empty, file logging is disabled.
* The **log directory path is fully customizable** via `LOG_FILE`. You must ensure the directory exists and is writable by the runtime user.

Systemd loads this file directly at runtime.

For `pda-dbinit`, only the `DB_*` and optional logging variables are required unless you pass `-dsn`.
You can also point it at an explicit env file with `-env-file /opt/pda-monitor/.env`.

---

## Local Development

### 1. Build

```bash
make build
make build-dbinit
```

Binary output:

```text
./bin/pda-monitor
./bin/pda-dbinit
```

---

### 2. Run Manually (Optional)

```bash
export $(cat .env | xargs)
./bin/pda-monitor
```

This is **not recommended for production**.

---

## User-Level systemd Deployment (Recommended)

This setup runs entirely under your user account — **no sudo required for the service itself**.

---

### 1. Install Binary

```bash
make install-user
```

This installs:

```text
~/.local/bin/pda-monitor
~/.local/bin/pda-dbinit
~/.config/pda-monitor/
~/.local/share/pda-monitor/
```

---

### 2. Create Runtime Env File

```bash
nano ~/.config/pda-monitor/.env
```

Paste your environment config there.

Initialize the database before starting the service:

```bash
~/.local/bin/pda-dbinit -env-file ~/.config/pda-monitor/.env
~/.local/bin/pda-dbinit -env-file ~/.config/pda-monitor/.env -create-user -username admin -password admin123 -role admin
```

---

### 3. Create systemd User Unit

```bash
mkdir -p ~/.config/systemd/user
nano ~/.config/systemd/user/pda-monitor.service
```

```ini
[Unit]
Description=PDA Telemetry Monitor (user service)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=%h/.config/pda-monitor/.env
ExecStart=%h/.local/bin/pda-monitor
WorkingDirectory=%h/.local/share/pda-monitor
Restart=always
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
```

---

### 4. Enable and Start

```bash
systemctl --user daemon-reload
systemctl --user enable --now pda-monitor.service
```

Follow logs:

```bash
journalctl --user -u pda-monitor -f
```

---

### 5. Run Without Active Login (Optional)

```bash
sudo loginctl enable-linger "$USER"
```

---

## Makefile Targets

```bash
make build         # Build binary to ./bin
make build-dbinit  # Build DB init CLI to ./bin
make install-user  # Install binary to ~/.local/bin
make redeploy-user # Pull latest GitHub changes, rebuild, install for current user, restart user service
make redeploy-system # Pull latest GitHub changes, rebuild, install under /opt, restart root service
make apply-system-dbinit # Run installed /opt/pda-monitor/pda-dbinit against /opt/pda-monitor/.env
make redeploy-system-db # System-wide redeploy, then apply dbinit patch, then restart service
make clean         # Remove build output
```

## Redeploy From GitHub

Use the redeploy script after the repository has already been cloned on the target machine.
It requires a clean working tree before pulling from GitHub.

User-level redeploy:

```bash
sh ./scripts/redeploy.sh --scope user
```

System-wide redeploy:

```bash
sh ./scripts/redeploy.sh --scope system
```

Deploy a specific branch or tag:

```bash
sh ./scripts/redeploy.sh --scope system --ref main
```

If Go is installed outside `PATH`, pass it explicitly:

```bash
GO=/usr/local/go/bin/go sh ./scripts/redeploy.sh --scope system
```

Useful options:

```bash
sh ./scripts/redeploy.sh --scope user --skip-pull      # deploy current checkout
sh ./scripts/redeploy.sh --scope system --skip-restart # install only
```

Apply schema/index updates on an existing system-wide install:

```bash
sh ./scripts/apply-system-dbinit.sh
```

System-wide redeploy plus dbinit patch:

```bash
sh ./scripts/redeploy-system-with-dbinit.sh
```

## Database Initialization CLI

Initialize the application tables inside an existing MySQL/MariaDB database:

```bash
./bin/pda-dbinit
```

Create an API user during setup:

```bash
./bin/pda-dbinit -create-user -username admin -password admin123 -role admin
```

Optional flags:

```bash
./bin/pda-dbinit -env-file /opt/pda-monitor/.env
./bin/pda-dbinit -dsn 'user:pass@tcp(localhost:3306)/pda_monitor?parseTime=true&loc=Asia%2FJakarta'
./bin/pda-dbinit -timeout 30s
```

For an already-installed root deployment under `/opt/pda-monitor`, prefer the helper script instead of invoking the CLI manually:

```bash
sh ./scripts/apply-system-dbinit.sh --env-file /opt/pda-monitor/.env
```

`-env-file` is preferred over putting DB credentials on the command line. Use `-dsn` only for one-off overrides.

This command creates these tables if they do not already exist:

```text
stations
formula_params
hourly_readings
users
station_alert_levels
```

It also applies additive schema/index updates used by the current backend performance path, including:

```text
hourly_readings(hour_bucket, nama_lokasi, recorded_at)
hourly_readings(recorded_at, nama_lokasi)
formula_params(nama_lokasi, priority, tma_min)
stations(sungai, nama_lokasi)
```

---

## Production Rules (Non-Negotiable)

* Never use `make run` in production.
* Never embed secrets in source code.
* Always run under systemd supervision.
* Always use `.env` for runtime injection.
* Never run Telegram tokens through CLI flags.

---

## Failure Handling

* Automatic restart on crash via systemd
* Journal-based logging for post-mortem analysis
* Environment isolation via `EnvironmentFile`

---

## Root-Level systemd Deployment (System-Wide)

Use this mode when the service must run as a **dedicated Linux service account**, start at boot, and survive user logouts without relying on `loginctl linger`.

This mode **requires sudo** and installs the binary under `/opt`.

---

### 1. Install Binary (System Path)

```bash
sudo mkdir -p /opt/pda-monitor
sudo cp ./bin/pda-monitor /opt/pda-monitor/pda-monitor
sudo chmod +x /opt/pda-monitor/pda-monitor
```

---

### 2. Create Runtime Env File

```bash
sudo nano /opt/pda-monitor/.env
```

Same `.env` structure as the user-level deployment.

---

### 3. (Recommended) Create Dedicated Service User

```bash
sudo useradd -r -s /usr/sbin/nologin pda
sudo chown -R pda:pda /opt/pda-monitor
```

This prevents the service from running as root.

---

### 4. Create Root systemd Unit

```bash
sudo nano /etc/systemd/system/pda-monitor.service
```

```ini
[Unit]
Description=PDA Telemetry Monitor (system service)
After=network.target mariadb.service
Wants=network.target

[Service]
Type=simple
User=pda
Group=pda
EnvironmentFile=/opt/pda-monitor/.env
ExecStart=/opt/pda-monitor/pda-monitor
WorkingDirectory=/opt/pda-monitor
Restart=always
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

---

### 5. Enable and Start (Root)

```bash
sudo systemctl daemon-reload
sudo systemctl enable pda-monitor
sudo systemctl start pda-monitor
```

Logs:

```bash
journalctl -u pda-monitor -f
```

---

## License

Internal project. No public license granted unless explicitly stated.
