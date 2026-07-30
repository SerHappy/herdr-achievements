#!/usr/bin/env sh
# Reproduces the terminal preview without requiring Herdr or persistent state.
set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
demo_dir=$(mktemp -d "${TMPDIR:-/tmp}/herdr-achievements-demo.XXXXXX")
trap 'rm -rf "$demo_dir"' EXIT HUP INT TERM

# The event command invokes Herdr for both reconciliation and notifications.
# A symlink back to this executable provides those two small fake commands
# without requiring a real Herdr installation.
case "${1:-}" in
	notification)
		exit 0
		;;
	api)
		if [ "${2:-}" = "snapshot" ]; then
			printf '%s\n' '{"result":{"snapshot":{"agents":[]}}}'
			exit 0
		fi
		;;
esac

cd "$root_dir"
go build -o "$demo_dir/herdr-achievements" ./cmd/herdr-achievements
ln -s "$root_dir/scripts/demo.sh" "$demo_dir/herdr"

emit_event() {
	HERDR_PLUGIN_EVENT=$1 \
	HERDR_PLUGIN_EVENT_JSON=$2 \
	HERDR_PLUGIN_STATE_DIR="$demo_dir/state" \
	HERDR_BIN_PATH="$demo_dir/herdr" \
	"$demo_dir/herdr-achievements" event
}

emit_event pane.agent_detected '{"event":"pane_agent_detected","data":{"type":"pane_agent_detected","pane_id":"demo:p1","workspace_id":"demo"}}'
emit_event pane.agent_status_changed '{"event":"pane_agent_status_changed","data":{"type":"pane_agent_status_changed","pane_id":"demo:p1","workspace_id":"demo","agent_status":"working"}}'
emit_event pane.agent_status_changed '{"event":"pane_agent_status_changed","data":{"type":"pane_agent_status_changed","pane_id":"demo:p1","workspace_id":"demo","agent_status":"done"}}'
emit_event pane.agent_status_changed '{"event":"pane_agent_status_changed","data":{"type":"pane_agent_status_changed","pane_id":"demo:p1","workspace_id":"demo","agent_status":"blocked"}}'
emit_event pane.agent_status_changed '{"event":"pane_agent_status_changed","data":{"type":"pane_agent_status_changed","pane_id":"demo:p1","workspace_id":"demo","agent_status":"working"}}'
emit_event pane.agent_status_changed '{"event":"pane_agent_status_changed","data":{"type":"pane_agent_status_changed","pane_id":"demo:p2","workspace_id":"demo","agent_status":"working"}}'
emit_event pane.agent_status_changed '{"event":"pane_agent_status_changed","data":{"type":"pane_agent_status_changed","pane_id":"demo:p3","workspace_id":"demo","agent_status":"working"}}'

HERDR_PLUGIN_STATE_DIR="$demo_dir/state" "$demo_dir/herdr-achievements" show
