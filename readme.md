# PDA Monitor Service

Lightweight Go-based telemetry monitor that ingests third-party telemetry data, exposes it via a REST API, sends Telegram notifications, and produces spreadsheet-style reports. Designed for Linux-first deployment with **user-level systemd supervision** and **`.env`-based runtime configuration**.

---

## Core Capabilities

* Telemetry data ingestion from external API
* REST API for downstream consumers
* Telegram bot notifications
* XLSX-style reporting output
* MariaDB/MySQL backend
* Fully `.env`-driven configuration
* User-level `systemd` service (no root required)

---

## Tech Stack

* **Language:** Go
* **Database:** MariaDB / MySQL
* **API Style:** REST
* **Messaging:** Telegram Bot API
* **Process Manager:** systemd (user-level)

---

## Project Layout

```text
.
├── cmd/server/        # Application entry point
├── internal/          # Core business logic
├── pkg/               # Shared libraries
├── bin/               # Local build output
├── Makefile
└── go.mod
```

---

## Runtime Environment Variables

All runtime configuration is supplied via a `.env` file.

Required variables:

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
```

Systemd loads this file directly at runtime.

---

## Local Development

### 1. Build

```bash
make build
```

Binary output:

```text
./bin/pda-monitor
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
~/.config/pda-monitor/
~/.local/share/pda-monitor/
```

---

### 2. Create Runtime Env File

```bash
nano ~/.config/pda-monitor/.env
```

Paste your environment config there.

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
make install-user  # Install binary to ~/.local/bin
make clean         # Remove build output
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