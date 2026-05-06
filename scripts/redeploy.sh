#!/bin/sh
set -eu

APP_NAME="pda-monitor"
DBINIT_NAME="pda-dbinit"
SERVICE_NAME="pda-monitor.service"
SYSTEM_INSTALL_DIR="/opt/pda-monitor"
SYSTEM_USER="pda"
GO_BIN="${GO:-go}"
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
if [ "$(id -u)" -eq 0 ]; then
	SCOPE="system"
else
	SCOPE="user"
fi
REF=""
SKIP_PULL="false"
SKIP_RESTART="false"

usage() {
	cat <<'USAGE'
Usage: scripts/redeploy.sh [--scope user|system] [--ref BRANCH_OR_TAG] [--skip-pull] [--skip-restart]

Redeploy the latest application changes from GitHub.

Options:
  --scope user       Install to the current user's ~/.local paths and restart systemd --user service.
  --scope system     Install system-wide under /opt/pda-monitor and restart root systemd service.
  --ref VALUE        Checkout/pull a specific branch or tag before building.
  --skip-pull        Build and deploy the current working tree without fetching from GitHub.
  --skip-restart     Install binaries without restarting systemd.
  -h, --help         Show this help.

Examples:
  sh scripts/redeploy.sh --scope user
  sh scripts/redeploy.sh --scope system --ref main
  GO=/usr/local/go/bin/go sh scripts/redeploy.sh --scope system
USAGE
}

log() {
	printf '[redeploy] %s\n' "$*"
}

fail() {
	printf '[redeploy] ERROR: %s\n' "$*" >&2
	exit 1
}

run_root() {
	if [ "$(id -u)" -eq 0 ]; then
		"$@"
	else
		sudo "$@"
	fi
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 is required but was not found in PATH"
}

check_prerequisites() {
	require_command git
	require_command make
	require_command install
	require_command systemctl

	if ! command -v "$GO_BIN" >/dev/null 2>&1; then
		fail "Go is required but '$GO_BIN' was not found. Install Go, add it to PATH, or run with GO=/path/to/go"
	fi
}

parse_args() {
	while [ "$#" -gt 0 ]; do
		case "$1" in
			--scope)
				[ "$#" -ge 2 ] || fail "--scope requires user or system"
				SCOPE="$2"
				shift 2
				;;
			--scope=*)
				SCOPE="${1#*=}"
				shift
				;;
			--ref)
				[ "$#" -ge 2 ] || fail "--ref requires a branch or tag"
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
			-h|--help)
				usage
				exit 0
				;;
			*)
				fail "unknown argument: $1"
				;;
		esac
	done

	case "$SCOPE" in
		user|system) ;;
		*) fail "--scope must be user or system" ;;
	esac
}

ensure_clean_tree() {
	git rev-parse --is-inside-work-tree >/dev/null 2>&1 || fail "$REPO_DIR is not a git repository"

	if ! git diff --quiet || ! git diff --cached --quiet; then
		fail "working tree has uncommitted changes; commit, stash, or rerun with --skip-pull"
	fi
}

update_from_github() {
	if [ "$SKIP_PULL" = "true" ]; then
		log "Skipping GitHub update; deploying current working tree"
		return
	fi

	ensure_clean_tree

	log "Fetching latest changes from GitHub"
	git fetch --prune origin

	if [ -n "$REF" ]; then
		log "Checking out ${REF}"
		git checkout "$REF"
	fi

	current_branch="$(git branch --show-current)"
	if [ -n "$current_branch" ]; then
		log "Pulling origin/${current_branch}"
		git pull --ff-only origin "$current_branch"
	else
		log "Detached HEAD; fetched refs only"
	fi
}

build_binaries() {
	log "Building ${APP_NAME} and ${DBINIT_NAME}"
	make GO="$GO_BIN" build-server build-dbinit
}

install_user() {
	log "Installing user-level binaries"
	mkdir -p "$HOME/.local/bin" "$HOME/.local/share/pda-monitor" "$HOME/.config/pda-monitor"
	install -m 0755 "$REPO_DIR/bin/$APP_NAME" "$HOME/.local/bin/$APP_NAME"
	install -m 0755 "$REPO_DIR/bin/$DBINIT_NAME" "$HOME/.local/bin/$DBINIT_NAME"

	if [ "$SKIP_RESTART" = "true" ]; then
		log "Skipping user service restart"
		return
	fi

	if systemctl --user cat "$SERVICE_NAME" >/dev/null 2>&1; then
		log "Reloading user systemd and restarting ${SERVICE_NAME}"
		systemctl --user daemon-reload
		systemctl --user restart "$SERVICE_NAME"
	else
		log "User service ${SERVICE_NAME} not found; install the unit before restarting"
	fi
}

install_system() {
	log "Installing system-wide binaries to ${SYSTEM_INSTALL_DIR}"
	run_root install -d -m 0755 "$SYSTEM_INSTALL_DIR"
	run_root install -m 0755 "$REPO_DIR/bin/$APP_NAME" "$SYSTEM_INSTALL_DIR/$APP_NAME"
	run_root install -m 0755 "$REPO_DIR/bin/$DBINIT_NAME" "$SYSTEM_INSTALL_DIR/$DBINIT_NAME"

	if id "$SYSTEM_USER" >/dev/null 2>&1; then
		run_root chown -R "$SYSTEM_USER:$SYSTEM_USER" "$SYSTEM_INSTALL_DIR"
	else
		log "Service user ${SYSTEM_USER} does not exist; leaving ${SYSTEM_INSTALL_DIR} owned by root"
	fi

	if [ "$SKIP_RESTART" = "true" ]; then
		log "Skipping system service restart"
		return
	fi

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
	cd "$REPO_DIR"

	check_prerequisites
	update_from_github
	build_binaries

	case "$SCOPE" in
		user) install_user ;;
		system) install_system ;;
	esac

	log "Redeploy complete"
}

main "$@"
