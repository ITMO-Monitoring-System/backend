#!/usr/bin/env python3
"""Poll the four GHCR :main tags and deploy changed services sequentially."""
import datetime as dt
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path('/opt/itmo-monitoring')
DEPLOY = ROOT / 'deploy'
CURRENT = DEPLOY / 'current' / 'release.json'
FAILED = DEPLOY / 'failed-updates.json'
MANAGER = DEPLOY / 'manage.py'
SERVICES = {
    'backend': 'ghcr.io/itmo-monitoring-system/backend',
    'face_recognizing': 'ghcr.io/itmo-monitoring-system/face-recognizing',
    'face_tracking': 'ghcr.io/itmo-monitoring-system/face-tracking',
    'frontend': 'ghcr.io/itmo-monitoring-system/frontend',
}


def run(args, *, capture=False):
    return subprocess.run(args, check=True, text=True, capture_output=capture)


def write_json(path, value):
    temporary = path.with_suffix('.tmp')
    temporary.write_text(json.dumps(value, indent=2) + '\n')
    temporary.chmod(0o600)
    temporary.replace(path)


def digest(service, repository):
    tag = repository + ':main'
    print(f'Checking {service}: {tag}', flush=True)
    run(['docker', 'pull', '--platform', 'linux/amd64', tag])
    info = json.loads(run(['docker', 'image', 'inspect', tag], capture=True).stdout)[0]
    if info.get('Architecture') != 'amd64' or info.get('Os') != 'linux':
        raise RuntimeError(f'{service}: expected linux/amd64 image')
    for value in info.get('RepoDigests', []):
        if value.split('@', 1)[0] == repository:
            return value
    raise RuntimeError(f'{service}: registry digest was not found')


def main():
    if not CURRENT.exists():
        raise RuntimeError('Initial deployment is not complete; automatic updates were not started.')
    current = json.loads(CURRENT.read_text())
    failed = json.loads(FAILED.read_text()) if FAILED.exists() else {}
    latest = {}
    for service, repository in SERVICES.items():
        latest[service] = digest(service, repository)

    blocked = []
    for service, record in list(failed.items()):
        if current.get(service) == latest[service] or record.get('digest') != latest[service]:
            failed.pop(service, None)
        else:
            blocked.append(service)
    write_json(FAILED, failed)
    if blocked:
        raise RuntimeError('Automatic updates blocked after a failed deploy: ' + ', '.join(blocked)
                           + '. Inspect logs, then use the recovery section of the local deployment runbook.')

    changed = [(service, image) for service, image in latest.items() if current.get(service) != image]
    if not changed:
        print('No new service images.')
        return
    for service, image in changed:
        print(f'Deploying changed service: {service}', flush=True)
        try:
            run(['python3', str(MANAGER), 'deploy-service', service, image])
        except subprocess.CalledProcessError as error:
            failed[service] = {
                'digest': image,
                'failed_at': dt.datetime.now(dt.timezone.utc).isoformat(),
                'exit_code': error.returncode,
            }
            write_json(FAILED, failed)
            raise RuntimeError(f'{service}: deployment failed; later automatic updates are blocked') from error
        current = json.loads(CURRENT.read_text())
        failed.pop(service, None)
        write_json(FAILED, failed)


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f'UPDATE ERROR: {error}', file=sys.stderr)
        sys.exit(1)
