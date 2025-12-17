#!/usr/bin/env bash
set -euo pipefail

# Re-run as root if needed (so we don't rely on sudo in the background)
if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  exec sudo -E "$0" "$@"
fi

OUTDIR="${1:-/tmp/ssh-strace}"
INTERVAL="${INTERVAL:-1}"
mkdir -p "$OUTDIR"

declare -A TRACED=()

echo "Running as root."
echo "Logging to: $OUTDIR"
echo "Polling every ${INTERVAL}s"

attach() {
  local pid="$1"
  local label="$2"
  local ts prefix
  ts="$(date +%Y%m%d-%H%M%S)"
  prefix="$OUTDIR/${ts}.${label}.pid${pid}"

  echo "[$(date --iso-8601=seconds)] attaching strace to $label PID $pid -> $prefix.*"

  # -ff follows forks; -o prefix will create prefix.<pid> files
  # Keep stderr visible so failures aren't silent.
  strace -ff -tt -s 256 -xx \
    -o "$prefix" \
    -p "$pid" \
    -e trace=process,network,read,write,sendto,recvfrom,connect,accept,openat,close,execve,getrandom \
    >/dev/null &
}

while true; do
  # Match both typical interactive session and the privsep helper if present
  # Examples:
  #   "sshd: user@pts/0"
  #   "sshd: user [priv]"
  while read -r pid args; do
    [[ -z "${pid:-}" ]] && continue
    [[ -n "${TRACED[$pid]:-}" ]] && continue

    TRACED["$pid"]=1
    attach "$pid" "sshd"
  done < <(ps -eo pid=,args= | awk '/[s]shd: .*(@pts\/| \[priv\])/{print $1, $0}')

  sleep "$INTERVAL"
done
