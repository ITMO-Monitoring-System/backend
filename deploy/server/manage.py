#!/usr/bin/env python3
"""Server deployment helper. Python standard library; never runs `down -v`."""
import argparse
import contextlib
import datetime as dt
import fcntl
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import subprocess
import sys
import urllib.request

ROOT = Path('/opt/itmo-monitoring')
DEPLOY = ROOT / 'deploy'
CONFIG = ROOT / 'configs'
HERE = Path(__file__).resolve().parent
SERVICES = ['backend', 'face_recognizing', 'face_tracking', 'frontend']
INFRA_SERVICES = ['db', 'rabbitmq-crops', 'rabbitmq-results', 'redis']
BASE_IMAGES = {
    'postgres': 'postgres:17-alpine',
    'rabbitmq': 'rabbitmq:3.13-management',
    'redis': 'redis:7-alpine',
    'migrate': 'migrate/migrate:v4.18.3',
}
REPOSITORIES = {
    'backend': 'ghcr.io/itmo-monitoring-system/backend',
    'frontend': 'ghcr.io/itmo-monitoring-system/frontend',
    'face_tracking': 'ghcr.io/itmo-monitoring-system/face-tracking',
    'face_recognizing': 'ghcr.io/itmo-monitoring-system/face-recognizing',
    'postgres': 'postgres', 'rabbitmq': 'rabbitmq',
    'redis': 'redis', 'migrate': 'migrate/migrate',
}


def run(args, *, capture=False, **kwargs):
    return subprocess.run(args, check=True, text=True, capture_output=capture, **kwargs)


def write(path, content, mode=0o600):
    path.write_text(content)
    path.chmod(mode)


def document(path):
    return json.loads(path.read_text())


def stamp():
    return dt.datetime.now(dt.timezone.utc).strftime('%Y%m%dT%H%M%SZ') + '-' + secrets.token_hex(3)


def link(name, target):
    temporary = DEPLOY / ('.' + name + '-' + secrets.token_hex(4))
    temporary.symlink_to(target)
    temporary.replace(DEPLOY / name)


def release_path(name='last-attempt'):
    path = DEPLOY / name
    if not path.exists():
        raise RuntimeError('No release yet. Run init, then deploy release.json.')
    return path.resolve()


def dc(path, *args, **kwargs):
    # Ignore ambient Compose interpolation variables from SSH/Actions sessions.
    env = dict(os.environ)
    for key in ['POSTGRES_PASSWORD', 'CROPS_PASSWORD', 'RESULTS_PASSWORD', 'RELEASE_DIR',
                *(name.upper() + '_IMAGE' for name in REPOSITORIES)]:
        env.pop(key, None)
    return run(['docker', 'compose', '-p', 'itmo-monitoring',
                '--env-file', str(CONFIG / 'runtime.env'),
                '--env-file', str(path / 'images.env'),
                '-f', str(path / 'compose.yaml'), *args], env=env, **kwargs)


@contextlib.contextmanager
def locked():
    if not DEPLOY.is_dir():
        raise RuntimeError('Create /opt/itmo-monitoring directories as instructed first.')
    with (DEPLOY / '.deploy.lock').open('a') as handle:
        try:
            fcntl.flock(handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as error:
            raise RuntimeError('Another deploy/backup/rollback is running. Wait for it.') from error
        yield


def init():
    for directory in [DEPLOY, CONFIG, ROOT / 'backups', ROOT / 'models']:
        if not directory.is_dir() or not os.access(directory, os.W_OK):
            raise RuntimeError(f'Missing/wrong ownership: {directory}')
    env_path = CONFIG / 'runtime.env'
    if env_path.exists():
        print('Existing credentials kept.')
        values = dict(line.split('=', 1) for line in env_path.read_text().splitlines() if line)
    else:
        volumes = run(['docker', 'volume', 'ls', '--filter', 'name=itmo-monitoring_',
                       '--format', '{{.Name}}'], capture=True).stdout.strip()
        if volumes:
            raise RuntimeError('Volumes already exist but runtime.env is missing. Restore configs; do not regenerate passwords.')
        values = {key: secrets.token_hex(32) for key in
                  ['POSTGRES_PASSWORD', 'CROPS_PASSWORD', 'RESULTS_PASSWORD', 'JWT_SECRET']}
        write(env_path, ''.join(f'{key}={value}\n' for key, value in values.items()))
    for key in ['POSTGRES_PASSWORD', 'CROPS_PASSWORD', 'RESULTS_PASSWORD', 'JWT_SECRET']:
        if not re.fullmatch(r'[a-f0-9]{64}', values.get(key, '')):
            raise RuntimeError(f'Unexpected format for {key}; keep the generated credentials.')
    backend = CONFIG / 'backend.toml'
    if not backend.exists():
        write(backend, f'''[app]
name = "monitoring_backend"
environment = "prod"
[http]
host = "0.0.0.0"
port = 8080
[postgres]
host = "db"
port = 5432
user = "monitoring"
password = "{values['POSTGRES_PASSWORD']}"
database = "monitoring"
sslmode = "disable"
max_conns = 20
[logger]
level = "info"
[rabbit]
ampq_url = "amqp://recognizing:{values['RESULTS_PASSWORD']}@recognizing-rabbitmq:5672/"
[jwt]
secret = "{values['JWT_SECRET']}"
ttl = "24h"
''', 0o644)  # Non-root appuser reads the single-file bind mount; host directory is 0700.
    (DEPLOY / 'releases').mkdir(exist_ok=True, mode=0o700)
    print('Configuration ready; passwords were not printed.')


def validate(path):
    images = document(path)
    if not isinstance(images, dict) or not set(SERVICES).issubset(images):
        raise RuntimeError('release.json must contain backend, frontend, face_tracking, face_recognizing.')
    if set(images) - set(REPOSITORIES):
        raise RuntimeError('Unknown image key in release.json.')
    for name, value in images.items():
        repo = REPOSITORIES[name]
        suffix = r'(?:@sha256:[a-f0-9]{64}|:sha-[a-f0-9]{40}|:main)' if name in SERVICES else r'@sha256:[a-f0-9]{64}'
        if not isinstance(value, str) or not re.fullmatch(re.escape(repo) + suffix, value):
            raise RuntimeError(f'Invalid {name} image. Use the expected repository and digest, sha-tag, or main tag for apps.')
    return images


def validate_service_image(service, image):
    if service not in SERVICES:
        raise RuntimeError('Unknown service: ' + service)
    expected = REPOSITORIES[service]
    if not re.fullmatch(re.escape(expected) + r'@sha256:[a-f0-9]{64}', image):
        raise RuntimeError(f'Invalid {service} image. The automatic updater must pass a full GHCR digest.')


def resolve(name, ref):
    print(f'Pulling {name}: {ref}', flush=True)
    run(['docker', 'pull', '--platform', 'linux/amd64', ref])
    info = json.loads(run(['docker', 'image', 'inspect', ref], capture=True).stdout)[0]
    if info['Architecture'] != 'amd64' or info['Os'] != 'linux':
        raise RuntimeError(f'{name}: expected linux/amd64 image.')
    expected = REPOSITORIES[name]
    for value in info.get('RepoDigests', []):
        normalized = value.removeprefix('docker.io/').removeprefix('library/')
        if normalized.split('@')[0] == expected:
            return normalized
    raise RuntimeError(f'{name}: could not resolve registry digest.')


def schema(path):
    text = dc(path, 'exec', '-T', 'db', 'psql', '-U', 'monitoring', '-d', 'monitoring',
              '-At', '-v', 'ON_ERROR_STOP=1', '-c',
              'SELECT version::text || \':\' || dirty::text FROM public.schema_migrations;', capture=True).stdout.strip()
    if not re.fullmatch(r'\d+:false', text):
        raise RuntimeError('Database migration state is dirty or missing. Inspect migrate logs; do not force the version.')
    return text


def backup(path):
    target = ROOT / 'backups' / ('postgres-' + stamp() + '.dump')
    with target.open('wb') as handle:
        dc(path, 'exec', '-T', 'db', 'pg_dump', '-U', 'monitoring', '-d', 'monitoring', '-Fc', stdout=handle)
    # Verify archive readability, not a substitute for a restore rehearsal.
    with target.open('rb') as handle:
        dc(path, 'exec', '-T', 'db', 'pg_restore', '--list', stdin=handle, stdout=subprocess.DEVNULL)
    print(f'Backup: {target}', flush=True)
    return target


def check(path):
    dc(path, 'ps', '-a')
    for service in SERVICES + INFRA_SERVICES:
        cid = dc(path, 'ps', '-q', service, capture=True).stdout.strip()
        if not cid:
            raise RuntimeError(f'{service} is not running.')
        status = run(['docker', 'inspect', '--format', '{{.State.Health.Status}}', cid], capture=True).stdout.strip()
        if status != 'healthy':
            raise RuntimeError(f'{service} is {status}.')
    schema(path)
    for endpoint in ['/api/health', '/tracking/health', '/fizon/']:
        with urllib.request.urlopen('http://127.0.0.1:18080' + endpoint, timeout=10) as response:
            if response.status != 200:
                raise RuntimeError(f'Unexpected HTTP status for {endpoint}')
    print('CONTAINERS AND HTTP OK. Perform the browser/camera acceptance checklist.', flush=True)


def deploy(manifest):
    images = validate(manifest)
    if not (CONFIG / 'runtime.env').exists() or not (CONFIG / 'backend.toml').exists():
        raise RuntimeError('Run init first.')
    infra = dict(BASE_IMAGES)
    if (DEPLOY / 'current/release.json').exists():
        current = document(DEPLOY / 'current/release.json')
        infra.update({key: current[key] for key in BASE_IMAGES})
    images = {**infra, **images}
    # Pull and validate everything before stopping any running application.
    resolved = {name: resolve(name, value) for name, value in images.items()}
    path = DEPLOY / 'releases' / stamp()
    path.mkdir(mode=0o700)
    shutil.copy2(HERE / 'compose.yaml', path / 'compose.yaml')
    shutil.copy2(HERE / 'nginx.conf', path / 'nginx.conf')
    shutil.copy2(HERE / 'manage.py', path / 'manage.py')
    write(path / 'release.json', json.dumps(resolved, indent=2) + '\n')
    write(path / 'images.env', ''.join(f'{name.upper()}_IMAGE={value}\n' for name, value in resolved.items())
          + f'RELEASE_DIR={path}\n')
    (path / 'migrations').mkdir(mode=0o755)
    run(['docker', 'run', '--rm', '--user', '0:0', '--entrypoint', '/bin/sh',
         '-v', str(path / 'migrations') + ':/out', resolved['backend'], '-ec',
         'cp /migrations/*.sql /out/ && chmod 644 /out/*.sql'])
    dc(path, 'config', '--quiet')
    link('last-attempt', path)
    write(path / 'status.txt', 'IN_PROGRESS\n')
    try:
        # Existing lectures must have been ended by the operator before deployment.
        dc(path, 'stop', '-t', '30', 'frontend', 'face_tracking', 'face_recognizing', 'backend')
        dc(path, 'up', '-d', '--wait', '--wait-timeout', '240', *INFRA_SERVICES)
        backup(path)
        dc(path, 'run', '--rm', '--no-deps', 'migrate')
        dc(path, 'run', '--rm', '--no-deps', 'model-preload')
        for service in SERVICES:
            dc(path, 'up', '-d', '--no-deps', '--force-recreate', '--wait', '--wait-timeout', '300', service)
        check(path)
        write(path / 'schema.txt', schema(path) + '\n')
        if (DEPLOY / 'current').exists():
            link('previous', (DEPLOY / 'current').resolve())
        link('current', path)
        write(path / 'status.txt', 'SUCCESS\n')
        print(f'DEPLOY SUCCESS: {path}')
    except BaseException:
        write(path / 'status.txt', 'FAILED_OR_INTERRUPTED\n')
        print(f'DEPLOY FAILED. Inspect {path} and logs. Services may be stopped; there is no automatic database rollback.', file=sys.stderr)
        raise


def deploy_service(service, image):
    """Replace one application image while restarting the four applications together."""
    validate_service_image(service, image)
    current = release_path('current')
    current_images = document(current / 'release.json')
    if not set(REPOSITORIES).issubset(current_images):
        raise RuntimeError('Current release is incomplete; run a full deploy first.')
    resolved_image = resolve(service, image)
    if current_images[service] == resolved_image:
        print(f'{service}: digest is already deployed.')
        return

    images = dict(current_images)
    images[service] = resolved_image
    path = DEPLOY / 'releases' / stamp()
    path.mkdir(mode=0o700)
    for name in ['compose.yaml', 'nginx.conf', 'manage.py']:
        shutil.copy2(HERE / name, path / name)
    write(path / 'release.json', json.dumps(images, indent=2) + '\n')
    write(path / 'images.env', ''.join(f'{name.upper()}_IMAGE={value}\n' for name, value in images.items())
          + f'RELEASE_DIR={path}\n')
    if service == 'backend':
        (path / 'migrations').mkdir(mode=0o755)
        run(['docker', 'run', '--rm', '--user', '0:0', '--entrypoint', '/bin/sh',
             '-v', str(path / 'migrations') + ':/out', resolved_image, '-ec',
             'cp /migrations/*.sql /out/ && chmod 644 /out/*.sql'])
    else:
        shutil.copytree(current / 'migrations', path / 'migrations')
    dc(path, 'config', '--quiet')
    link('last-attempt', path)
    write(path / 'status.txt', 'IN_PROGRESS\n')
    try:
        dc(path, 'stop', '-t', '30', 'frontend', 'face_tracking', 'face_recognizing', 'backend')
        if service == 'backend':
            backup(path)
            dc(path, 'run', '--rm', '--no-deps', 'migrate')
        if service == 'face_recognizing':
            dc(path, 'run', '--rm', '--no-deps', 'model-preload')
        for name in SERVICES:
            dc(path, 'up', '-d', '--no-deps', '--force-recreate', '--wait', '--wait-timeout', '300', name)
        check(path)
        write(path / 'schema.txt', schema(path) + '\n')
        link('previous', current)
        link('current', path)
        write(path / 'status.txt', 'SUCCESS\n')
        print(f'DEPLOY SERVICE SUCCESS: {service} -> {resolved_image}')
    except BaseException:
        write(path / 'status.txt', 'FAILED_OR_INTERRUPTED\n')
        print(f'DEPLOY SERVICE FAILED: {service}. Automatic updates are blocked until this digest is handled.',
              file=sys.stderr)
        raise


def rollback():
    attempt = release_path()
    # On failed deploy current is still last known-good; after success use previous.
    target = release_path('current' if (attempt / 'status.txt').read_text().strip() != 'SUCCESS' else 'previous')
    if schema(attempt) != (target / 'schema.txt').read_text().strip():
        raise RuntimeError('Schema changed since the target release. Image-only rollback refused. See database recovery instructions.')
    dc(attempt, 'stop', '-t', '30', 'frontend', 'face_tracking', 'face_recognizing', 'backend')
    backup(attempt)
    for service in SERVICES:
        dc(target, 'up', '-d', '--no-deps', '--force-recreate', '--wait', '--wait-timeout', '300', service)
    check(target)
    link('last-attempt', target)
    link('current', target)
    print(f'ROLLED BACK APPLICATION IMAGES: {target}. Database was not restored.')


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='command', required=True)
    for name in ['init', 'check', 'backup', 'rollback', 'status']:
        sub.add_parser(name)
    for name in ['deploy', 'validate']:
        sub.add_parser(name).add_argument('manifest', type=Path)
    service_parser = sub.add_parser('deploy-service')
    service_parser.add_argument('service', choices=SERVICES)
    service_parser.add_argument('image')
    sub.add_parser('compose').add_argument('args', nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if args.command == 'validate':
        validate(args.manifest)
        print('Release manifest format OK (registry access is checked during deploy).')
        return
    if args.command in ['init', 'deploy', 'deploy-service', 'backup', 'rollback']:
        with locked():
            if args.command == 'init': init()
            elif args.command == 'deploy': deploy(args.manifest)
            elif args.command == 'deploy-service': deploy_service(args.service, args.image)
            elif args.command == 'backup': backup(release_path())
            else: rollback()
    elif args.command == 'check': check(release_path())
    elif args.command == 'status':
        path = release_path()
        print(f'Last attempt: {path}; state: {(path / "status.txt").read_text().strip()}')
        dc(path, 'ps', '-a')
    else:
        if not args.args:
            raise RuntimeError('Specify Compose arguments, e.g. compose logs --tail 100 backend')
        dc(release_path(), *args.args)


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, OSError, ValueError, subprocess.CalledProcessError) as error:
        # Avoid echoing failed command arguments: migration connection strings contain secrets.
        if isinstance(error, subprocess.CalledProcessError):
            print(f'Command failed (exit {error.returncode}). See output above and the local deployment runbook.', file=sys.stderr)
        else:
            print(f'ERROR: {error}', file=sys.stderr)
        sys.exit(1)
