#!/usr/bin/env bash
set -Eeuo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo 'Run: sudo bash install-server.sh' >&2
  exit 1
fi
kit_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
project_dir=/opt/itmo-monitoring

if [ -e "$project_dir/deploy/manage.py" ] || [ -e "$project_dir/configs/runtime.env" ]; then
  echo 'An installation already exists. Continue with manage.py; this installer does not overwrite it.' >&2
  exit 1
fi
docker info >/dev/null
docker compose version
if ! id itmo-deploy >/dev/null 2>&1; then
  useradd --create-home --user-group --shell /bin/bash itmo-deploy
fi
usermod -aG docker itmo-deploy
install -d -m 0750 "$project_dir"
chown glass-room-1:itmo-deploy "$project_dir"
chmod 0750 "$project_dir"
for name in deploy configs backups models lecture-logs lecture-log-control; do
  install -d -o itmo-deploy -g itmo-deploy -m 0700 "$project_dir/$name"
done
for name in manage.py watch_updates.py compose.yaml nginx.conf release.json itmo-backup.service itmo-backup.timer itmo-update.service itmo-update.timer lecture_logs.py; do
  install -o itmo-deploy -g itmo-deploy -m 0600 "$kit_dir/$name" "$project_dir/deploy/$name"
done
install -o itmo-deploy -g itmo-deploy -m 0600 "$kit_dir/lecture-logs.example.json" "$project_dir/deploy/lecture-logs.json"
install -m 0644 "$kit_dir/itmo-lecture-logs.service" /etc/systemd/system/itmo-lecture-logs.service
usermod -aG systemd-journal itmo-deploy
systemctl daemon-reload
systemctl enable --now itmo-lecture-logs.service
sudo -H -u itmo-deploy docker info >/dev/null
echo 'INSTALL COMPLETE. Review journald host settings; next: registry login, manage.py init, manage.py deploy.'
