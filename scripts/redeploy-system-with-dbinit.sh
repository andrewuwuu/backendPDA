#!/bin/sh
set -eu

SERVICE_NAME="pda-monitor.service"
SYSTEM_INSTALL_DIR="/opt/pda-monitor"
ENV_FILE=""
TIMEOUT=""
REF=""
SKIP_PULL="false"
SKIP_RESTART="false"
GO_BIN="${GO:-go}"
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

usage() {
	cat <<'USAGE'
Usage: scripts/redeploy-system-with-dbinit.sh [--ref BRANCH_OR_TAG] [--skip-pull] [--skip-restart] [--install-dir DIR] [--env-file PATH] [--timeout DURATION]

Redeploy the system-wide installation, apply pda-dbinit using the installed
binary and production env file, then restart the root systemd service.

Options:
  --ref VALUE        Checkout/pull a specific branch or tag before building.
  --skip-pull        Deploy the current working tree without fetching from GitHub.
  --skip-restart     Install binaries and apply dbinit, but do not restart systemd.
  --install-dir DIR  System install directory. Default: /opt/pda-monitor
  --env-file PATH    Runtime env file. Default: <install-dir>/.env
  --timeout DURATION Override dbinit timeout, e.g. 30s
  -h, --help         Show this help.

Examples:
  sh scripts/redeploy-system-with-dbinit.sh
  sh scripts/redeploy-system-with-dbinit.sh --ref main
  GO=/usr/local/go/bin/go sh scripts/redeploy-system-with-dbinit.sh --skip-pull
USAGE
}

log() {
	printf '[redeploy-system-with-dbinit] %s\n' "$*"
}

fail() {
	printf '[redeploy-system-with-dbinit] ERROR: %s\n' "$*" >&2
	exit 1
}

run_root() {
	if [ "$(id -u)" -eq 0 ]; then
		"$@"
	else
		sudo "$@"
	fi
}

parse_args() {
	while [ "$#" -gt 0 ]; do
		case "$1" in
			--ref)
				[ "$#" -ge 2 ] || fail "--ref requires a value"
				REF="$2"
				shift 2
				;;
			--ref=*)
				REF="${1#*=}"
				shift
				;;
			--skip-pull)
				SKIP_PULL="true"
				shift
				;;
			--skip-restart)
				SKIP_RESTART="true"
				shift
				;;
			--install-dir)
				[ "$#" -ge 2 ] || fail "--install-dir requires a value"
				SYSTEM_INSTALL_DIR="$2"
				shift 2
				;;
			--install-dir=*)
				SYSTEM_INSTALL_DIR="${1#*=}"
				shift
				;;
			--env-file)
				[ "$#" -ge 2 ] || fail "--env-file requires a value"
				ENV_FILE="$2"
				shift 2
				;;
			--env-file=*)
				ENV_FILE="${1#*=}"
				shift
				;;
			--timeout)
				[ "$#" -ge 2 ] || fail "--timeout requires a value"
				TIMEOUT="$2"
				shift 2
				;;
			--timeout=*)
				TIMEOUT="${1#*=}"
				shift
				;;
			-h|--help)
				usage
				exit 0
				;;
			*)
				fail "unknown argument: $1"
				;;
		esac
	done
}

restart_service() {
	if run_root systemctl cat "$SERVICE_NAME" >/dev/null 2>&1; then
		log "Reloading root systemd and restarting ${SERVICE_NAME}"
		run_root systemctl daemon-reload
		run_root systemctl restart "$SERVICE_NAME"
	else
		log "System service ${SERVICE_NAME} not found; install the unit before restarting"
	fi
}

main() {
	parse_args "$@"

	log "Installing updated system-wide binaries without restarting yet"
	if [ "$SKIP_PULL" = "true" ] && [ -n "$REF" ]; then
		GO="$GO_BIN" sh "$SCRIPT_DIR/redeploy.sh" --scope system --skip-pull --skip-restart --ref "$REF"
	elif [ "$SKIP_PULL" = "true" ]; then
		GO="$GO_BIN" sh "$SCRIPT_DIR/redeploy.sh" --scope system --skip-pull --skip-restart
	elif [ -n "$REF" ]; then
		GO="$GO_BIN" sh "$SCRIPT_DIR/redeploy.sh" --scope system --skip-restart --ref "$REF"
	else
		GO="$GO_BIN" sh "$SCRIPT_DIR/redeploy.sh" --scope system --skip-restart
	fi

	log "Applying installed database/schema patch"
	if [ -n "$ENV_FILE" ] && [ -n "$TIMEOUT" ]; then
		sh "$SCRIPT_DIR/apply-system-dbinit.sh" --install-dir "$SYSTEM_INSTALL_DIR" --env-file "$ENV_FILE" --timeout "$TIMEOUT"
	elif [ -n "$ENV_FILE" ]; then
		sh "$SCRIPT_DIR/apply-system-dbinit.sh" --install-dir "$SYSTEM_INSTALL_DIR" --env-file "$ENV_FILE"
	elif [ -n "$TIMEOUT" ]; then
		sh "$SCRIPT_DIR/apply-system-dbinit.sh" --install-dir "$SYSTEM_INSTALL_DIR" --timeout "$TIMEOUT"
	else
		sh "$SCRIPT_DIR/apply-system-dbinit.sh" --install-dir "$SYSTEM_INSTALL_DIR"
	fi

	if [ "$SKIP_RESTART" = "true" ]; then
		log "Skipping service restart"
		exit 0
	fi

	restart_service
	log "System-wide redeploy with dbinit complete"
}

main "$@"
