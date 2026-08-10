#!/usr/bin/env bash
# Starts, stops and restarts the local SaveForge documentation portal from any
# working directory.
#
# Two processes are managed together:
#
#   * Scalar Docs, served from backend/swagger/docs-portal on localhost:7970.
#     This is the documentation portal and the interface a user opens.
#   * backend/swagger on 127.0.0.1:8788. This is not a user interface; it exists
#     only so the "Try it" button in the Scalar API Reference has a local API
#     host to call, and it stays bound to loopback.
#
# The script owns exactly the two processes it started itself, recorded in its
# own state directory. It never searches for a process by name or by port, and
# it never signals anything it has not first verified as its own, so a Swagger
# or Scalar process started by any other means is invisible to it and is left
# untouched.
#
#   scripts/run_swagger.sh start [swagger flags...]
#   scripts/run_swagger.sh stop
#   scripts/run_swagger.sh restart [swagger flags...]
set -euo pipefail

# The repository root comes from the script's own location, so Go module
# resolution and the catalog data path never depend on $PWD. pwd -P resolves a
# symlinked scripts/ directory to the real one.
script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repository_root="$(cd -- "${script_directory}/.." && pwd -P)"

readonly default_address="127.0.0.1:8788"
readonly catalog_data_directory="${repository_root}/backend/gamecatalog/data"
readonly portal_directory="${repository_root}/backend/swagger/docs-portal"
readonly portal_configuration="scalar.config.json"
readonly portal_port="7970"
# The Scalar preview server binds the loopback name, and on this platform that
# resolves to ::1 only, so 127.0.0.1 cannot reach it. Every check below uses the
# same name the printed URLs use.
readonly portal_url="http://localhost:${portal_port}/"
readonly scalar_package="@scalar/cli@2.0.1"
# The Scalar preview downloads its package on first use and builds the whole
# portal before it listens, so it is given a longer budget than the Go server.
readonly readiness_attempts=60
readonly portal_readiness_attempts=180
readonly stop_attempts=40

# State lives outside the working tree so `start` never dirties `git status`,
# and is keyed by the repository path so two checkouts manage separate servers.
# The portal log in particular must stay out of docs-portal: the preview watches
# that directory and a log written inside it triggers an endless rebuild loop.
state_key="$(printf '%s' "${repository_root}" | cksum | cut -d ' ' -f 1)"
readonly state_directory="${XDG_STATE_HOME:-${HOME}/.local/state}/saveforge/swagger/${state_key}"
readonly pid_file="${state_directory}/pid"
readonly arguments_file="${state_directory}/arguments"
readonly binary_file="${state_directory}/swagger"
readonly log_file="${state_directory}/log"
readonly portal_pid_file="${state_directory}/portal-pid"
readonly portal_log_file="${state_directory}/portal-log"

fail() {
	printf 'run_swagger: %s\n' "$1" >&2
	exit 1
}

# read_persisted_arguments fills swagger_arguments from the NUL-delimited file
# written by the previous start. The file is read as plain data and never
# sourced or word-split, so an argument can hold any shell metacharacter without
# ever being executed.
read_persisted_arguments() {
	swagger_arguments=()
	[ -f "${arguments_file}" ] || return 0
	local argument
	while IFS= read -r -d '' argument; do
		swagger_arguments+=("${argument}")
	done <"${arguments_file}"
}

write_persisted_arguments() {
	if [ "$#" -eq 0 ]; then
		: >"${arguments_file}"
		return 0
	fi
	printf '%s\0' "$@" >"${arguments_file}"
}

# effective_address reports the address the API host was told to bind, so the
# readiness probe checks reality instead of repeating the default. Both
# -addr VALUE and -addr=VALUE are recognised, and the last occurrence wins,
# which is the rule Go's flag package applies.
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

# owned_pid prints the PID of the running API host this script started, and
# fails for anything else. Liveness alone is not enough: a PID is reused, so the
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

# owned_portal_pid prints the PID of the running Scalar preview this script
# started, and fails for anything else.
#
# The preview is not one process. `npx` execs npm, npm spawns the scalar
# launcher, and the launcher spawns the process that actually listens. A
# measured experiment showed that a SIGTERM sent to the recorded PID alone kills
# the first two and leaves the listener orphaned onto init, so the whole process
# group has to be signalled instead. That is only safe if the group is known to
# contain nothing else, so `start` makes the preview a group leader of a fresh
# group and this check refuses to accept a PID that is not still the leader of
# its own group. Together with the recorded command line, that rules out both a
# reused PID and a group that has since been joined by a foreign process.
owned_portal_pid() {
	[ -f "${portal_pid_file}" ] || return 1

	local recorded_pid
	IFS= read -r recorded_pid <"${portal_pid_file}" || return 1
	case "${recorded_pid}" in
	'' | *[!0-9]*)
		return 1
		;;
	esac

	kill -0 "${recorded_pid}" 2>/dev/null || return 1

	local group_id
	group_id="$(ps -p "${recorded_pid}" -o pgid= 2>/dev/null | tr -d ' ')" || return 1
	[ "${group_id}" = "${recorded_pid}" ] || return 1

	local command_line
	command_line="$(ps -p "${recorded_pid}" -o command= 2>/dev/null)" || return 1
	case "${command_line}" in
	*"${scalar_package} project preview ${portal_configuration}"*) ;;
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

discard_stale_portal_state() {
	rm -f "${portal_pid_file}"
}

# start_api_host builds and starts the loopback API host the "Try it" button
# calls. It is an internal dependency, so it prints progress but no URL.
start_api_host() {
	local address
	address="$(effective_address "$@")"
	local probe
	probe="$(probe_address "${address}")"

	# A readiness probe cannot tell this script's server apart from someone
	# else's on the same address, and the Go command loads the catalog before it
	# binds, so a doomed start stays alive long enough to be mistaken for ready.
	# Asking the address first removes that ambiguity. It is a single request to
	# the address the caller named, never a scan, and the answer is only ever
	# used to refuse to start: nothing found this way is signalled or stopped.
	if curl -fsS -o /dev/null --max-time 2 "http://${probe}/healthz" 2>/dev/null; then
		fail "address ${address} is already served by a process this script does not manage; it was left running. Stop it yourself or start on another -addr"
	fi

	# Building first means the PID recorded below belongs to the server itself.
	# `go run` would put its own supervising process there, and stopping that
	# leaves the real server orphaned.
	printf 'Building the local API host ...\n'
	(cd -- "${repository_root}" && go build -o "${binary_file}" ./backend/swagger) ||
		fail "go build failed"

	# Recorded only once the start is actually attempted, so a refused start
	# leaves the arguments a later `restart` reuses untouched.
	write_persisted_arguments "$@"

	printf '\n=== start %s ===\n' "$(date '+%Y-%m-%d %H:%M:%S')" >>"${log_file}"
	# The default -data is passed first so a -data supplied by the caller comes
	# later on the command line and wins.
	"${binary_file}" -data "${catalog_data_directory}" "$@" >>"${log_file}" 2>&1 &
	local server_pid=$!
	printf '%s\n' "${server_pid}" >"${pid_file}"

	local attempt=0
	while [ "${attempt}" -lt "${readiness_attempts}" ]; do
		# Liveness is checked first: a server that already exited, on a failed
		# bind or anything else, will never become ready, and a probe answered
		# after that death was answered by somebody else.
		kill -0 "${server_pid}" 2>/dev/null || break
		if curl -fsS -o /dev/null --max-time 2 "http://${probe}/healthz" 2>/dev/null &&
			kill -0 "${server_pid}" 2>/dev/null; then
			return 0
		fi
		sleep 0.5
		attempt=$((attempt + 1))
	done

	# The start failed. Anything still alive here is the process this script
	# just spawned and recorded, so it is verified before being signalled. The
	# stop runs in a subshell so its own error path cannot pre-empt the
	# diagnosis below.
	(stop_api_host) >/dev/null 2>&1 || true
	discard_stale_state

	if grep -q 'address already in use' "${log_file}" 2>/dev/null; then
		fail "address ${address} is already in use by a process this script does not manage; it was left running. See ${log_file}"
	fi
	fail "the local API host did not become ready on http://${probe}/healthz. See ${log_file}"
}

# check_portal_preconditions refuses a start the portal cannot survive. It runs
# before anything is spawned, so a portal that was never going to come up does
# not leave a half-started API host behind. As with the API host, the single
# request to the portal address is only ever used to refuse to start: a process
# found this way is never signalled.
check_portal_preconditions() {
	[ -f "${portal_directory}/${portal_configuration}" ] ||
		fail "missing ${portal_directory}/${portal_configuration}"
	command -v npx >/dev/null 2>&1 || fail "npx is required to run Scalar Docs but was not found"

	if curl -fsS -o /dev/null --max-time 2 "${portal_url}" 2>/dev/null; then
		fail "${portal_url} is already served by a process this script does not manage; it was left running. Stop it yourself before starting the portal"
	fi
}

# start_portal starts the Scalar Docs preview that serves the portal itself.
start_portal() {
	printf 'Starting Scalar Docs ...\n'
	printf '\n=== start %s ===\n' "$(date '+%Y-%m-%d %H:%M:%S')" >>"${portal_log_file}"

	# Job control puts the preview into a process group of its own whose leader
	# is the PID recorded here, which is what makes a group-wide stop safe. exec
	# keeps that leader as the npx process itself rather than an extra subshell.
	# stdin is closed so the preview cannot block on a prompt, and the log is
	# written outside the watched directory.
	set -m
	(
		cd -- "${portal_directory}" &&
			exec npx --yes "${scalar_package}" project preview "${portal_configuration}" \
				--port "${portal_port}" --no-open
	) </dev/null >>"${portal_log_file}" 2>&1 &
	local preview_pid=$!
	set +m
	printf '%s\n' "${preview_pid}" >"${portal_pid_file}"

	local attempt=0
	while [ "${attempt}" -lt "${portal_readiness_attempts}" ]; do
		kill -0 "${preview_pid}" 2>/dev/null || break
		if curl -fsS -o /dev/null --max-time 2 "${portal_url}" 2>/dev/null &&
			kill -0 "${preview_pid}" 2>/dev/null; then
			return 0
		fi
		sleep 0.5
		attempt=$((attempt + 1))
	done

	# The portal is useless without its API host, and a half-started pair is
	# worse than none, so a failed portal takes down both processes this call
	# started. Only processes verified as ours are signalled.
	(stop_portal) >/dev/null 2>&1 || true
	discard_stale_portal_state
	(stop_api_host) >/dev/null 2>&1 || true
	fail "Scalar Docs did not become ready on ${portal_url}. See ${portal_log_file}"
}

stop_api_host() {
	local running_pid
	if ! running_pid="$(owned_pid)"; then
		discard_stale_state
		return 0
	fi

	kill -TERM "${running_pid}" 2>/dev/null || true

	local attempt=0
	while [ "${attempt}" -lt "${stop_attempts}" ]; do
		if ! owned_pid >/dev/null; then
			discard_stale_state
			printf 'Stopped the local API host (pid %s).\n' "${running_pid}"
			return 0
		fi
		sleep 0.25
		attempt=$((attempt + 1))
	done

	fail "the local API host (pid ${running_pid}) did not exit within $((stop_attempts / 4)) seconds after SIGTERM; it was left running. See ${log_file}"
}

# stop_portal signals the verified process group, not just the recorded PID,
# because the process that actually listens is a grandchild that survives a
# PID-only SIGTERM and reparents onto init.
stop_portal() {
	local running_pid
	if ! running_pid="$(owned_portal_pid)"; then
		discard_stale_portal_state
		return 0
	fi

	kill -TERM "-${running_pid}" 2>/dev/null || true

	local attempt=0
	while [ "${attempt}" -lt "${stop_attempts}" ]; do
		# The group is gone once no process still reports it, which covers the
		# children as well as the recorded leader.
		if ! ps -A -o pgid= 2>/dev/null | tr -d ' ' | grep -qx "${running_pid}"; then
			discard_stale_portal_state
			printf 'Stopped Scalar Docs (process group %s).\n' "${running_pid}"
			return 0
		fi
		sleep 0.25
		attempt=$((attempt + 1))
	done

	fail "Scalar Docs (process group ${running_pid}) did not exit within $((stop_attempts / 4)) seconds after SIGTERM; it was left running. See ${portal_log_file}"
}

command_start() {
	mkdir -p "${state_directory}"

	local running_pid
	if running_pid="$(owned_pid)"; then
		fail "a local API host started by this script is already running (pid ${running_pid}); use restart or stop"
	fi
	if running_pid="$(owned_portal_pid)"; then
		fail "Scalar Docs started by this script is already running (pid ${running_pid}); use restart or stop"
	fi
	discard_stale_state
	discard_stale_portal_state

	check_portal_preconditions
	start_api_host "$@"
	start_portal

	printf '\n'
	printf 'Scalar Docs: %s\n' "${portal_url}"
	printf 'API Reference: http://localhost:%s/api\n' "${portal_port}"
}

command_stop() {
	local anything_running=0
	if owned_portal_pid >/dev/null || owned_pid >/dev/null; then
		anything_running=1
	fi

	stop_portal
	stop_api_host

	[ "${anything_running}" -eq 1 ] ||
		printf 'No process started by this script is running; nothing to stop.\n'
}

command_restart() {
	mkdir -p "${state_directory}"

	local swagger_arguments
	if [ "$#" -gt 0 ]; then
		# Supplied flags replace the persisted ones, for this run and the next.
		swagger_arguments=("$@")
	else
		read_persisted_arguments
	fi

	command_stop
	command_start ${swagger_arguments[@]+"${swagger_arguments[@]}"}
}

usage() {
	printf 'usage: %s start [swagger flags...] | stop | restart [swagger flags...]\n' "$0" >&2
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
