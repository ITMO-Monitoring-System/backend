#!/usr/bin/env bash
set -Eeuo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo 'Run as root on the server.' >&2
  exit 1
fi
kit_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
project_dir=/opt/itmo-monitoring
if [ ! -e "$project_dir/deploy/manage.py" ]; then
  echo 'Existing installation not found; use install-server.sh.' >&2
  exit 1
fi
id itmo-deploy >/dev/null
timer_was_active=false
upgrade_ok=false
if systemctl is-active --quiet itmo-update.timer; then
  timer_was_active=true
  systemctl stop itmo-update.timer
fi
restore_timer() {
  if [ "$timer_was_active" = true ] && [ "$upgrade_ok" = false ]; then
    echo 'Upgrade failed; itmo-update.timer remains stopped for inspection.' >&2
  fi
}
trap restore_timer EXIT
install -d -o itmo-deploy -g itmo-deploy -m 0700 "$project_dir/lecture-logs"
install -d -o itmo-deploy -g itmo-deploy -m 0700 "$project_dir/lecture-log-control"
backup_dir="$project_dir/backups/lecture-logging-$(date -u +%Y%m%dT%H%M%SZ)"
install -d -o itmo-deploy -g itmo-deploy -m 0700 "$backup_dir"
for name in compose.yaml nginx.conf; do
  if [ -e "$project_dir/deploy/$name" ]; then
    cp -p "$project_dir/deploy/$name" "$backup_dir/$name"
  fi
  install -o itmo-deploy -g itmo-deploy -m 0600 "$kit_dir/$name" "$project_dir/deploy/$name"
done
if [ -e "$project_dir/deploy/lecture_logs.py" ]; then
  cp -p "$project_dir/deploy/lecture_logs.py" "$backup_dir/lecture_logs.py"
fi
if [ -e /etc/systemd/system/itmo-lecture-logs.service ]; then
  cp -p /etc/systemd/system/itmo-lecture-logs.service "$backup_dir/itmo-lecture-logs.service"
fi
install -o itmo-deploy -g itmo-deploy -m 0600 "$kit_dir/lecture_logs.py" "$project_dir/deploy/lecture_logs.py"
if [ ! -e "$project_dir/deploy/lecture-logs.json" ]; then
  install -o itmo-deploy -g itmo-deploy -m 0600 "$kit_dir/lecture-logs.example.json" "$project_dir/deploy/lecture-logs.json"
fi
python3 - "$project_dir/deploy/lecture-logs.json" <<'PY'
import json, os, sys
path = sys.argv[1]
owner = os.stat(path)
with open(path, encoding="utf-8") as file:
    config = json.load(file)
config.setdefault("control_socket", "/opt/itmo-monitoring/lecture-log-control/control.sock")
temp = path + ".tmp"
with open(temp, "w", encoding="utf-8") as file:
    os.fchmod(file.fileno(), 0o600)
    os.fchown(file.fileno(), owner.st_uid, owner.st_gid)
    json.dump(config, file, ensure_ascii=False, indent=2)
    file.write("\n")
os.replace(temp, path)
PY
install -m 0644 "$kit_dir/itmo-lecture-logs.service" /etc/systemd/system/itmo-lecture-logs.service
usermod -aG systemd-journal itmo-deploy
systemctl daemon-reload
systemctl enable --now itmo-lecture-logs.service
systemctl restart itmo-lecture-logs.service
upgrade_ok=true
echo 'Collector installed. Review journald settings, redeploy Compose outside a lecture, then restore the prior itmo-update.timer state.'
if [ "$timer_was_active" = true ]; then
  echo 'The itmo-update.timer was active and is intentionally stopped until deployment checks pass.'
fi
