#!/bin/sh
set -eu

APP_NAME="pda-monitor"
DBINIT_NAME="pda-dbinit"
SYSTEM_INSTALL_DIR="/opt/pda-monitor"
ENV_FILE=""
TIMEOUT=""

usage() {
	cat <<'USAGE'
Usage: scripts/apply-system-dbinit.sh [--install-dir DIR] [--env-file PATH] [--timeout DURATION]

Run the installed system-wide pda-dbinit binary against the production .env file
to apply additive schema/index changes on an existing deployment.

Options:
  --install-dir DIR   System install directory. Default: /opt/pda-monitor
  --env-file PATH     Runtime env file. Default: <install-dir>/.env
  --timeout DURATION  Override dbinit timeout, e.g. 30s
  -h, --help          Show this help.

Examples:
  sh scripts/apply-system-dbinit.sh
  sh scripts/apply-system-dbinit.sh --env-file /opt/pda-monitor/.env
  sh scripts/apply-system-dbinit.sh --timeout 30s
USAGE
}

log() {
	printf '[apply-system-dbinit] %s\n' "$*"
}

fail() {
	printf '[apply-system-dbinit] ERROR: %s\n' "$*" >&2
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

main() {
	parse_args "$@"

	if [ -z "$ENV_FILE" ]; then
		ENV_FILE="$SYSTEM_INSTALL_DIR/.env"
	fi

	DBINIT_PATH="$SYSTEM_INSTALL_DIR/$DBINIT_NAME"

	[ -x "$DBINIT_PATH" ] || fail "installed dbinit binary not found or not executable: $DBINIT_PATH"
	[ -f "$ENV_FILE" ] || fail "env file not found: $ENV_FILE"

	log "Running $DBINIT_PATH with env file $ENV_FILE"
	if [ -n "$TIMEOUT" ]; then
		run_root "$DBINIT_PATH" -env-file "$ENV_FILE" -timeout "$TIMEOUT"
	else
		run_root "$DBINIT_PATH" -env-file "$ENV_FILE"
	fi

	log "Schema/index patch complete for ${APP_NAME}"
}

main "$@"
