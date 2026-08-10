#!/usr/bin/env bash
# Starts, stops and restarts the local GameCatalog DB Viewer from any working
# directory.
#
# The viewer is a read-only browser for the catalog documents; it binds
# 127.0.0.1:8787 by default and modifies neither GameCatalog nor save files.
#
# The script owns exactly the one process it started itself, recorded in its own
# state directory. It never searches for a process by name or by port, and it
# never signals anything it has not first verified as its own, so a viewer
# started by any other means is invisible to it and is left untouched.
#
#   scripts/run_viewer.sh start [viewer flags...]
#   scripts/run_viewer.sh stop
#   scripts/run_viewer.sh restart [viewer flags...]
set -euo pipefail

# The repository root comes from the script's own location, so Go module
# resolution and the catalog data path never depend on $PWD. pwd -P resolves a
# symlinked scripts/ directory to the real one.
script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repository_root="$(cd -- "${script_directory}/.." && pwd -P)"

readonly default_address="127.0.0.1:8787"
readonly catalog_data_directory="${repository_root}/backend/gamecatalog/data"
readonly readiness_attempts=60
readonly stop_attempts=40

# State lives outside the working tree so `start` never dirties `git status`,
# and is keyed by the repository path so two checkouts manage separate viewers.
# The directory is distinct from the Swagger script's state, so neither script
# can ever read or signal the other's process.
state_key="$(printf '%s' "${repository_root}" | cksum | cut -d ' ' -f 1)"
readonly state_directory="${XDG_STATE_HOME:-${HOME}/.local/state}/saveforge/gamecatalog-viewer/${state_key}"
readonly pid_file="${state_directory}/pid"
readonly arguments_file="${state_directory}/arguments"
readonly binary_file="${state_directory}/gamecatalog-viewer"
readonly log_file="${state_directory}/log"

fail() {
	printf 'run_viewer: %s\n' "$1" >&2
	exit 1
}

# read_persisted_arguments fills viewer_arguments from the NUL-delimited file
# written by the previous start. The file is read as plain data and never
# sourced or word-split, so an argument can hold any shell metacharacter without
# ever being executed.
read_persisted_arguments() {
	viewer_arguments=()
	[ -f "${arguments_file}" ] || return 0
	local argument
	while IFS= read -r -d '' argument; do
		viewer_arguments+=("${argument}")
	done <"${arguments_file}"
}

write_persisted_arguments() {
	if [ "$#" -eq 0 ]; then
		: >"${arguments_file}"
		return 0
	fi
	printf '%s\0' "$@" >"${arguments_file}"
}

# effective_address reports the address the viewer was told to bind, so the
# readiness probe and the printed URL state reality instead of repeating the
# default. Both -addr VALUE and -addr=VALUE are recognised, and the last
# occurrence wins, which is the rule Go's flag package applies.
effective_address() {
	local address="${default_address}"
	while [ "$#" -gt 0 ]; do
		case "$1" in
		-addr=* | --addr=*)
			address="${1#*=}"
			;;
		-addr | --addr)
			# A trailing -addr with no value is left to the Go command to
			# reject; the probed address stays the default until then.
			if [ "$#" -gt 1 ]; then
				address="$2"
				shift
			fi
			;;
		esac
		shift
	done
	printf '%s' "${address}"
}

# probe_address turns a bind address into one a local HTTP check can reach. A
# wildcard bind answers on loopback, but cannot be connected to as written.
probe_address() {
	local address="$1"
	local host="${address%:*}"
	local port="${address##*:}"
	case "${host}" in
	'' | '0.0.0.0' | '[::]' | '*')
		host="127.0.0.1"
		;;
	esac
	printf '%s:%s' "${host}" "${port}"
}

# owned_pid prints the PID of the running viewer this script started, and fails
# for anything else. Liveness alone is not enough: a PID is reused, so the
# process command has to point at the binary in this script's state directory
# before the PID is ever signalled.
owned_pid() {
	[ -f "${pid_file}" ] || return 1

	local recorded_pid
	IFS= read -r recorded_pid <"${pid_file}" || return 1
	case "${recorded_pid}" in
	'' | *[!0-9]*)
		return 1
		;;
	esac

	kill -0 "${recorded_pid}" 2>/dev/null || return 1

	local command_line
	command_line="$(ps -p "${recorded_pid}" -o command= 2>/dev/null)" || return 1
	case "${command_line}" in
	"${binary_file}" | "${binary_file} "*) ;;
	*)
		return 1
		;;
	esac

	printf '%s' "${recorded_pid}"
}

# discard_stale_state removes the record of a process that is gone or was never
# ours. It only deletes this script's own files and signals nothing.
discard_stale_state() {
	rm -f "${pid_file}"
}

start_viewer() {
	local address
	address="$(effective_address "$@")"
	local probe
	probe="$(probe_address "${address}")"

	# A readiness probe cannot tell this script's viewer apart from someone
	# else's on the same address, and the Go command loads the catalog before it
	# binds, so a doomed start stays alive long enough to be mistaken for ready.
	# Asking the address first removes that ambiguity. It is a single request to
	# the address the caller named, never a scan, and the answer is only ever
	# used to refuse to start: nothing found this way is signalled or stopped.
	if curl -fsS -o /dev/null --max-time 2 "http://${probe}/healthz" 2>/dev/null; then
		fail "address ${address} is already served by a process this script does not manage; it was left running. Stop it yourself or start on another -addr"
	fi

	# Building first means the PID recorded below belongs to the viewer itself.
	# `go run` would put its own supervising process there, and stopping that
	# leaves the real viewer orphaned.
	printf 'Building the GameCatalog DB Viewer ...\n'
	(cd -- "${repository_root}" && go build -o "${binary_file}" ./backend/gamecatalog/dbviewer/cmd/gamecatalog-viewer) ||
		fail "go build failed"

	# Recorded only once the start is actually attempted, so a refused start
	# leaves the arguments a later `restart` reuses untouched.
	write_persisted_arguments "$@"

	printf '\n=== start %s ===\n' "$(date '+%Y-%m-%d %H:%M:%S')" >>"${log_file}"
	# The default -data is passed first so a -data supplied by the caller comes
	# later on the command line and wins.
	"${binary_file}" -data "${catalog_data_directory}" "$@" >>"${log_file}" 2>&1 &
	local viewer_pid=$!
	printf '%s\n' "${viewer_pid}" >"${pid_file}"

	local attempt=0
	while [ "${attempt}" -lt "${readiness_attempts}" ]; do
		# Liveness is checked first: a viewer that already exited, on a failed
		# bind or anything else, will never become ready, and a probe answered
		# after that death was answered by somebody else.
		kill -0 "${viewer_pid}" 2>/dev/null || break
		if curl -fsS -o /dev/null --max-time 2 "http://${probe}/healthz" 2>/dev/null &&
			kill -0 "${viewer_pid}" 2>/dev/null; then
			printf '\n'
			printf 'GameCatalog DB Viewer: http://%s/\n' "${address}"
			return 0
		fi
		sleep 0.5
		attempt=$((attempt + 1))
	done

	# The start failed. Anything still alive here is the process this script
	# just spawned and recorded, so it is verified before being signalled. The
	# stop runs in a subshell so its own error path cannot pre-empt the
	# diagnosis below.
	(stop_viewer) >/dev/null 2>&1 || true
	discard_stale_state

	if grep -q 'address already in use' "${log_file}" 2>/dev/null; then
		fail "address ${address} is already in use by a process this script does not manage; it was left running. See ${log_file}"
	fi
	fail "the GameCatalog DB Viewer did not become ready on http://${probe}/healthz. See ${log_file}"
}

stop_viewer() {
	local running_pid
	if ! running_pid="$(owned_pid)"; then
		discard_stale_state
		return 1
	fi

	kill -TERM "${running_pid}" 2>/dev/null || true

	local attempt=0
	while [ "${attempt}" -lt "${stop_attempts}" ]; do
		if ! owned_pid >/dev/null; then
			discard_stale_state
			printf 'Stopped the GameCatalog DB Viewer (pid %s).\n' "${running_pid}"
			return 0
		fi
		sleep 0.25
		attempt=$((attempt + 1))
	done

	fail "the GameCatalog DB Viewer (pid ${running_pid}) did not exit within $((stop_attempts / 4)) seconds after SIGTERM; it was left running. See ${log_file}"
}

command_start() {
	mkdir -p "${state_directory}"

	local running_pid
	if running_pid="$(owned_pid)"; then
		fail "a GameCatalog DB Viewer started by this script is already running (pid ${running_pid}); use restart or stop"
	fi
	discard_stale_state

	start_viewer "$@"
}

command_stop() {
	stop_viewer ||
		printf 'No process started by this script is running; nothing to stop.\n'
}

command_restart() {
	mkdir -p "${state_directory}"

	local viewer_arguments
	if [ "$#" -gt 0 ]; then
		# Supplied flags replace the persisted ones, for this run and the next.
		viewer_arguments=("$@")
	else
		read_persisted_arguments
	fi

	command_stop
	command_start ${viewer_arguments[@]+"${viewer_arguments[@]}"}
}

usage() {
	printf 'usage: %s start [viewer flags...] | stop | restart [viewer flags...]\n' "$0" >&2
	exit 2
}

[ "$#" -gt 0 ] || usage
subcommand="$1"
shift

case "${subcommand}" in
start)
	command_start "$@"
	;;
stop)
	[ "$#" -eq 0 ] || fail "stop takes no arguments"
	command_stop
	;;
restart)
	command_restart "$@"
	;;
*)
	usage
	;;
esac
