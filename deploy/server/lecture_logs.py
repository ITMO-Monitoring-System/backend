#!/usr/bin/env python3
"""Collect bounded lecture archives from Docker journald and selected system units."""

from __future__ import annotations

import argparse
import contextlib
import datetime as dt
import hashlib
import json
import os
import re
import shutil
import socket
import socketserver
import sqlite3
import subprocess
import sys
import threading
import time
import uuid
from pathlib import Path

UTC = dt.timezone.utc
SERVICES = ("frontend", "backend", "face_tracking", "face_recognizing", "db",
            "rabbitmq-crops", "rabbitmq-results", "redis", "migrate", "model-preload")
UNITS = {"docker.service": "docker", "containerd.service": "containerd",
         "itmo-update.service": "itmo-update", "itmo-backup.service": "itmo-backup",
         "itmo-lecture-logs.service": "collector", "systemd-journald.service": "journald"}
INSTALLED_CONFIG = Path(__file__).with_name("lecture-logs.json")
DEFAULT_CONFIG = INSTALLED_CONFIG if INSTALLED_CONFIG.exists() else Path(__file__).with_name("lecture-logs.example.json")


def stamp(value=None):
    return (value or dt.datetime.now(UTC)).isoformat(timespec="microseconds").replace("+00:00", "Z")


def parse_stamp(value):
    return dt.datetime.fromisoformat(value.replace("Z", "+00:00"))


def timestamp_us(entry):
    try:
        return int(entry.get("__REALTIME_TIMESTAMP", 0))
    except (TypeError, ValueError):
        return 0


def atomic_json(path, value):
    temp = path.with_name(path.name + ".tmp")
    with temp.open("w", encoding="utf-8") as file:
        os.fchmod(file.fileno(), 0o600)
        json.dump(value, file, ensure_ascii=False, indent=2, sort_keys=True)
        file.write("\n")
        file.flush()
        os.fsync(file.fileno())
    os.replace(temp, path)
    sync_dir(path.parent)


def sync_dir(path):
    fd = os.open(path, os.O_RDONLY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def config(path):
    cfg = json.loads(Path(path).read_text(encoding="utf-8"))
    cfg["root"] = Path(cfg["root"])
    if "control_socket" in cfg:
        cfg["control_socket"] = Path(cfg["control_socket"])
    for key in ("pre_roll_seconds", "post_roll_seconds", "poll_seconds", "min_free_bytes"):
        if not isinstance(cfg.get(key), int) or cfg[key] < 0:
            raise ValueError(f"invalid {key} in collector configuration")
    if cfg["poll_seconds"] == 0:
        raise ValueError("poll_seconds must be positive")
    return cfg


@contextlib.contextmanager
def locked(root):
    import fcntl

    root.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(root, 0o700)
    with (root / ".collector.lock").open("a") as file:
        os.fchmod(file.fileno(), 0o600)
        fcntl.flock(file, fcntl.LOCK_EX)
        yield


def sessions(root):
    for path in sorted(root.iterdir()):
        if path.is_dir() and ((path / "state.json").exists() or (path / "manifest.json").exists()):
            yield path


def read_state(path):
    state_path = path / "state.json"
    return json.loads((state_path if state_path.exists() else path / "manifest.json").read_text(encoding="utf-8"))


def save_state(path, state):
    atomic_json(path / "state.json", state)


def find_session(root, session_id):
    if not re.fullmatch(r"[0-9a-f]{12}", session_id):
        raise ValueError("invalid session ID")
    found = [p for p in sessions(root) if read_state(p)["session_id"] == session_id]
    if len(found) != 1:
        raise ValueError("session not found")
    return found[0]


def run(*args, timeout=30):
    return subprocess.run(args, text=True, capture_output=True, timeout=timeout, check=False)


def journal_cmd():
    return os.environ.get("LECTURE_LOGS_JOURNALCTL", "journalctl")


def preflight(cfg):
    result = run(journal_cmd(), "--no-pager", "--all", "-o", "json", "-n", "1")
    if result.returncode or not result.stdout.strip():
        raise RuntimeError("system journal is unavailable to this account")
    if shutil.disk_usage(cfg["root"]).free < cfg["min_free_bytes"]:
        raise RuntimeError("insufficient free space for a new archive")
    if cfg.get("require_service", True):
        result = run("systemctl", "is-active", "--quiet", "itmo-lecture-logs.service")
        if result.returncode:
            raise RuntimeError("itmo-lecture-logs.service is not active")
    if cfg.get("require_containers", True):
        snapshot = container_snapshot()
        if "error" in snapshot:
            raise RuntimeError(snapshot["error"])
        missing = [name for name in SERVICES[:8] if snapshot.get(name, {}).get("state") != "running"]
        if missing:
            raise RuntimeError("project containers not running: " + ", ".join(missing))


def release():
    path = Path("/opt/itmo-monitoring/deploy/current/release.json")
    try:
        data = json.loads(path.read_text())
        return {key: data[key] for key in ("backend", "frontend", "face_tracking", "face_recognizing") if key in data}
    except (OSError, ValueError):
        return None


def container_snapshot():
    result = run("docker", "ps", "-a", "--filter", "label=com.docker.compose.project=itmo-monitoring",
                 "--format", "{{json .}}")
    if result.returncode:
        return {"error": "Docker container status unavailable"}
    found = {}
    for line in result.stdout.splitlines():
        try:
            item = json.loads(line)
            name = item.get("Names", "")
            for service in SERVICES:
                if name == f"itmo-monitoring-{service}-1" or name == f"itmo-monitoring_{service}_1":
                    found[service] = {"id": item.get("ID"), "image": item.get("Image"), "state": item.get("State")}
        except ValueError:
            pass
    return found


@contextlib.contextmanager
def db(path):
    conn = sqlite3.connect(path / "events.sqlite3", timeout=10)
    os.chmod(path / "events.sqlite3", 0o600)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=FULL")
    conn.execute("CREATE TABLE IF NOT EXISTS events (cursor TEXT PRIMARY KEY, ts INTEGER NOT NULL, "
                 "source TEXT NOT NULL, kind TEXT NOT NULL, container_id TEXT, image TEXT, "
                 "boot_id TEXT, monotonic TEXT, message TEXT NOT NULL, lecture_id TEXT, request_id TEXT)")
    conn.execute("CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)")
    try:
        with conn:
            yield conn
    finally:
        conn.close()


def meta(conn, key):
    row = conn.execute("SELECT value FROM meta WHERE key=?", (key,)).fetchone()
    return row[0] if row else None


def set_meta(conn, key, value):
    conn.execute("INSERT OR REPLACE INTO meta VALUES (?, ?)", (key, value))


SECRET_PATTERNS = (
    re.compile(r"(?i)\b(amqps?|postgres(?:ql)?|redis|rtsps?|https?)://([^\s/@:]+):([^\s/@]+)@"),
    re.compile(r"(?i)\b(authorization[\"']?\s*[:=]\s*[\"']?(?:bearer|basic)\s+)[^\s,;\"']+"),
    re.compile(r"(?i)\b((?:password|passwd|secret|token|jwt|api[_-]?key|image_b64)[\"']?\s*[:=]\s*[\"']?)[^\s,;\"']+"),
    re.compile(r"\beyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b"),
)
SENSITIVE_KEYS = {"authorization", "password", "passwd", "secret", "token", "jwt",
                  "access_token", "refresh_token", "api_key", "image_b64", "image",
                  "embedding", "embeddings", "face_image", "photo"}
HTTP_REQUEST_PATTERN = re.compile(r"(?i)\b(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s+/[^\s\"]+(?=\s+HTTP/\d)")


def scrub_json(value):
    if isinstance(value, dict):
        output, count = {}, 0
        for key, item in value.items():
            if str(key).lower() in SENSITIVE_KEYS:
                output[key] = "[REDACTED]"
                count += 1
            else:
                output[key], n = scrub_json(item)
                count += n
        return output, count
    if isinstance(value, list):
        output, count = [], 0
        for item in value:
            cleaned, n = scrub_json(item)
            output.append(cleaned)
            count += n
        return output, count
    return value, 0


def redact(value):
    text = str(value)
    text, request_count = HTTP_REQUEST_PATTERN.subn(lambda m: m.group(1) + " [REDACTED_PATH]", text)
    text, count = SECRET_PATTERNS[0].subn(lambda m: f"{m.group(1)}://[REDACTED]@", text)
    for pattern in SECRET_PATTERNS[1:3]:
        text, n = pattern.subn(lambda m: m.group(1) + "[REDACTED]", text)
        count += n
    text, n = SECRET_PATTERNS[3].subn("[REDACTED_JWT]", text)
    return text, count + n + request_count


def message_value(entry):
    message = entry.get("MESSAGE", "")
    if isinstance(message, list):
        message = bytes(message).decode("utf-8", "replace")
    count = 0
    if isinstance(message, str) and message.startswith("{"):
        try:
            cleaned, count = scrub_json(json.loads(message))
            if count:
                message = json.dumps(cleaned, ensure_ascii=False)
        except ValueError:
            pass
    message, n = redact(message)
    return message, count + n


def classify(entry):
    tag = str(entry.get("CONTAINER_TAG", ""))
    if tag.startswith("itmo-monitoring/"):
        service = tag.split("/", 1)[1]
        if service in SERVICES:
            return "service", service
    unit = entry.get("_SYSTEMD_UNIT")
    if unit in UNITS and "CONTAINER_ID_FULL" not in entry and "CONTAINER_ID" not in entry:
        return "system", UNITS[unit]
    if "CONTAINER_ID_FULL" not in entry and "CONTAINER_ID" not in entry and str(entry.get("SYSLOG_IDENTIFIER", "")) == "systemd-journald":
        return "system", "journald"
    return None, None


def event_ids(message):
    try:
        app = json.loads(message)
        if isinstance(app, dict):
            lecture = app.get("lecture_id")
            request = app.get("request_id")
            return (str(lecture) if lecture is not None else None,
                    str(request) if request is not None else None)
    except ValueError:
        pass
    match = re.search(r"(?<![\w])(?:lecture_id|lecture)=(\d+)(?!\d)", message)
    return (match.group(1) if match else None, None)


def cursor_available(cursor):
    result = run(journal_cmd(), "--no-pager", "--all", "-o", "json", "--cursor", cursor, "-n", "+1")
    if result.returncode:
        return False
    for line in result.stdout.splitlines():
        try:
            return json.loads(line).get("__CURSOR") == cursor
        except ValueError:
            pass
    return False


def mark_gap(path, state, reason):
    if state["known_gaps"] and state["known_gaps"][-1].get("reason") == reason and state["known_gaps"][-1].get("last_cursor") == state.get("last_cursor"):
        return
    gap = {"detected_at": stamp(), "reason": reason, "last_cursor": state.get("last_cursor"),
           "last_event_at": state.get("last_event_at"), "resumed_at": None}
    state["known_gaps"].append(gap)
    state["warnings"].append(reason)
    save_state(path, state)


def collect(path, state, cfg):
    if state["state"] not in ("active", "finalizing", "interrupted"):
        return
    with db(path) as conn:
        cursor = meta(conn, "cursor")
        lost_cursor = bool(cursor and not cursor_available(cursor))
        if lost_cursor:
            mark_gap(path, state, "journal cursor no longer available; resumed from last event timestamp")
            cursor = None
            since = int(meta(conn, "last_timestamp") or int(parse_stamp(state["capture_from"]).timestamp() * 1_000_000)) // 1_000_000
        else:
            since = int(parse_stamp(state["capture_from"]).timestamp())
        cmd = [journal_cmd(), "--no-pager", "--all", "-o", "json"]
        cmd += ["--after-cursor", cursor] if cursor else ["--since", f"@{since}"]
        until = state.get("capture_to")
        if until:
            cmd += ["--until", f"@{int(parse_stamp(until).timestamp()) + 1}"]
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        last_cursor = cursor
        last_ts = None
        first_ts = None
        assert process.stdout is not None
        try:
            for line in process.stdout:
                try:
                    entry = json.loads(line)
                except ValueError:
                    continue
                event_cursor = entry.get("__CURSOR")
                event_ts = timestamp_us(entry)
                if not event_cursor:
                    continue
                if event_ts < int(parse_stamp(state["capture_from"]).timestamp() * 1_000_000):
                    continue
                if until and event_ts > int(parse_stamp(until).timestamp() * 1_000_000):
                    continue
                if first_ts is None:
                    first_ts = event_ts
                kind, source = classify(entry)
                if kind:
                    msg, masked = message_value(entry)
                    if source == "journald" and re.search(r"(?i)\b(suppressed|dropped)\s+\d+\s+messages?\b", msg):
                        mark_gap(path, state, "journald reported suppressed or dropped messages")
                    lecture, request = event_ids(msg)
                    inserted = conn.execute("INSERT OR IGNORE INTO events VALUES (?,?,?,?,?,?,?,?,?,?,?)",
                                 (event_cursor, event_ts, source, kind,
                                  str(entry.get("CONTAINER_ID_FULL") or entry.get("CONTAINER_ID") or ""),
                                  str(entry.get("IMAGE_NAME") or ""), str(entry.get("_BOOT_ID") or ""),
                                  str(entry.get("__MONOTONIC_TIMESTAMP") or ""), msg, lecture, request))
                    if masked and inserted.rowcount:
                        set_meta(conn, "redacted_records", str(int(meta(conn, "redacted_records") or 0) + 1))
                last_cursor, last_ts = event_cursor, event_ts
            stderr = process.stderr.read() if process.stderr else ""
            code = process.wait()
            if code:
                raise RuntimeError("journalctl failed: " + redact(stderr[:300])[0])
            if last_cursor and last_ts is not None:
                set_meta(conn, "cursor", last_cursor)
                set_meta(conn, "last_timestamp", str(last_ts))
        except Exception:
            process.kill()
            process.wait()
            raise
        finally:
            process.stdout.close()
            if process.stderr:
                process.stderr.close()
    if last_cursor and last_ts is not None:
        if lost_cursor and state["known_gaps"]:
            state["known_gaps"][-1]["resumed_at"] = stamp(dt.datetime.fromtimestamp(first_ts / 1_000_000, UTC))
        previous = state.get("last_event_at")
        if previous and last_ts < int(parse_stamp(previous).timestamp() * 1_000_000):
            state["warnings"].append("journal realtime moved backwards; check host clock synchronization")
        state["last_cursor"] = last_cursor
        state["last_event_at"] = stamp(dt.datetime.fromtimestamp(last_ts / 1_000_000, UTC))
        save_state(path, state)


def format_event(row):
    cursor, ts, source, kind, cid, image, boot, mono, message, lecture, request = row
    event = {"timestamp": stamp(dt.datetime.fromtimestamp(ts / 1_000_000, UTC)),
             "source": source, "kind": kind, "container_id": cid or None,
             "image": image or None, "boot_id": boot or None, "monotonic_timestamp": mono or None,
             "lecture_id": lecture, "request_id": request, "message": message}
    return json.dumps(event, ensure_ascii=False, separators=(",", ":")) + "\n"


def write_lines(path, lines):
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    temp = path.with_name(path.name + ".tmp")
    with temp.open("w", encoding="utf-8") as file:
        os.fchmod(file.fileno(), 0o600)
        file.writelines(lines)
        file.flush()
        os.fsync(file.fileno())
    os.replace(temp, path)
    sync_dir(path.parent)


def sha256_file(path):
    digest = hashlib.sha256()
    with path.open("rb") as file:
        for chunk in iter(lambda: file.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def cleanup_internal(path):
    for name in ("events.sqlite3", "events.sqlite3-wal", "events.sqlite3-shm", "state.done"):
        (path / name).unlink(missing_ok=True)
    sync_dir(path)


def has_internal(path):
    return any((path / name).exists() for name in
               ("events.sqlite3", "events.sqlite3-wal", "events.sqlite3-shm", "state.done"))


def finalize(path, state):
    if state["state"] in ("completed", "partial"):
        return
    state["state"] = "finalizing"
    save_state(path, state)
    counts = {"all": 0, "services": {name: 0 for name in SERVICES},
              "system": {name: 0 for name in UNITS.values()}, "lecture_only": 0}
    containers = {}
    writers = {}

    def writer(relative):
        target = path / relative
        if target not in writers:
            target.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            temporary = target.with_name(target.name + ".tmp")
            file = temporary.open("w", encoding="utf-8")
            os.fchmod(file.fileno(), 0o600)
            writers[target] = (temporary, file)
        return writers[target][1]

    writer(Path("all.log"))
    writer(Path("lecture-only.log"))
    for name in SERVICES[:8]:
        writer(Path("services") / (name + ".log"))
    try:
        with db(path) as conn:
            redacted_count = int(meta(conn, "redacted_records") or 0)
            for row in conn.execute("SELECT * FROM events ORDER BY ts, cursor"):
                _, _, source, kind, cid, image, *_ = row
                line = format_event(row)
                writer(Path("all.log")).write(line)
                counts["all"] += 1
                group = "services" if kind == "service" else "system"
                writer(Path(group) / (source + ".log")).write(line)
                counts[group][source] += 1
                if row[-2] == str(state["lecture_id"]):
                    writer(Path("lecture-only.log")).write(line)
                    counts["lecture_only"] += 1
                if cid:
                    containers[cid] = {"service": source, "image": image}
        for target, (temporary, file) in writers.items():
            file.flush()
            os.fsync(file.fileno())
            file.close()
            os.replace(temporary, target)
            sync_dir(target.parent)
    finally:
        for _, file in writers.values():
            if not file.closed:
                file.close()
    at_start = state.get("containers_at_start") or {}
    at_end = container_snapshot()
    if state.get("source_check_enabled", True) and "error" in at_end:
        state["known_gaps"].append({"reason": "Docker container status unavailable at end",
                                    "detected_at": stamp()})
    source_status = {}
    for name in SERVICES[:8]:
        count = counts["services"][name]
        source_status[name] = {"records": count,
                               "start_state": at_start.get(name, {}).get("state"),
                               "end_state": at_end.get(name, {}).get("state")}
        if state.get("source_check_enabled", True) and count == 0 and at_start.get(name, {}).get("state") != "running" and at_end.get(name, {}).get("state") != "running":
            state["known_gaps"].append({"reason": f"{name} container was not observed during capture",
                                        "detected_at": stamp()})
    state.update(state="partial" if state["known_gaps"] else "completed", containers_seen=containers,
                 record_counts=counts, redaction={"applied": True, "records_affected": redacted_count},
                 release_at_end=release(), containers_at_end=at_end, source_status=source_status,
                 sources={"services": list(SERVICES), "system_units": list(UNITS)})
    manifest = {key: value for key, value in state.items() if key != "last_cursor"}
    atomic_json(path / "manifest.json", manifest)
    checksum_lines = []
    for file in sorted(path.rglob("*")):
        if file.is_file() and file.name not in ("checksums.sha256", "events.sqlite3", "events.sqlite3-wal",
                                                  "events.sqlite3-shm", "state.json", ".collector.lock") and not file.name.endswith(".tmp"):
            checksum_lines.append(f"{sha256_file(file)}  {file.relative_to(path)}\n")
    write_lines(path / "checksums.sha256", checksum_lines)
    os.replace(path / "state.json", path / "state.done")
    sync_dir(path)
    cleanup_internal(path)


def create_session(cfg, lecture_id):
    if not re.fullmatch(r"[0-9]+", lecture_id):
        raise ValueError("lecture ID must be a nonnegative integer")
    preflight(cfg)
    now = dt.datetime.now(UTC)
    sid = uuid.uuid4().hex[:12]
    path = cfg["root"] / (now.strftime("%Y-%m-%dT%H-%M-%SZ") + f"_lecture-{lecture_id}_run-{sid}")
    path.mkdir(mode=0o700)
    state = {"schema_version": 1, "session_id": sid, "lecture_id": lecture_id, "state": "active",
             "started_at": stamp(now), "ended_at": None,
             "capture_from": stamp(now - dt.timedelta(seconds=cfg["pre_roll_seconds"])), "capture_to": None,
             "pre_roll_seconds": cfg["pre_roll_seconds"], "post_roll_seconds": cfg["post_roll_seconds"],
             "release_at_start": release(), "release_at_end": None, "containers_at_start": container_snapshot(),
             "warnings": [], "known_gaps": [], "overlaps_with": [],
             "source_check_enabled": cfg.get("require_containers", True), "browser_logs_collected": False,
             "external_proxy_logs_collected": False, "limitations": ["browser and external proxy logs are not collected",
             "only application-emitted events can be archived"]}
    for existing_path in sessions(cfg["root"]):
        previous = read_state(existing_path)
        if previous["state"] in ("active", "finalizing", "interrupted") and (
            not previous.get("capture_to") or parse_stamp(previous["capture_to"]) >= parse_stamp(state["capture_from"])
        ):
            state["overlaps_with"].append(previous["session_id"])
            previous.setdefault("overlaps_with", []).append(sid)
            save_state(existing_path, previous)
    save_state(path, state)
    with db(path):
        pass
    return {"session_id": sid, "path": str(path), "capture_from": state["capture_from"], "state": "active"}


def start(cfg, lecture_id):
    print(json.dumps(create_session(cfg, lecture_id)))


def stop_session(path, state):
    if state["state"] in ("completed", "partial"):
        if has_internal(path):
            cleanup_internal(path)
        return {"session_id": state["session_id"], "state": state["state"], "path": str(path)}
    if state["ended_at"] is None:
        now = dt.datetime.now(UTC)
        state["ended_at"] = stamp(now)
        state["capture_to"] = stamp(now + dt.timedelta(seconds=state["post_roll_seconds"]))
        state["state"] = "finalizing"
        save_state(path, state)
    return {"session_id": state["session_id"], "state": state["state"],
            "capture_to": state["capture_to"], "path": str(path)}


def stop(path, state):
    print(json.dumps(stop_session(path, state)))


def control(cfg, request):
    lecture_id = str(request.get("lecture_id", ""))
    if not re.fullmatch(r"[0-9]+", lecture_id):
        raise ValueError("invalid lecture ID")
    action = request.get("action")
    if action not in ("start", "stop"):
        raise ValueError("invalid control action")
    with locked(cfg["root"]):
        matching = [(path, state) for path in sessions(cfg["root"])
                    if (state := read_state(path))["lecture_id"] == lecture_id]
        matching.sort(key=lambda item: item[1]["started_at"])
        active = [(path, state) for path, state in matching
                  if state["state"] in ("active", "interrupted") and state.get("ended_at") is None]
        if action == "start":
            if active:
                path, state = active[-1]
                if state["state"] != "active":
                    raise RuntimeError("lecture archive is interrupted")
                return {"session_id": state["session_id"], "path": str(path),
                        "capture_from": state["capture_from"], "state": state["state"]}
            return create_session(cfg, lecture_id)
        if active:
            path, state = active[-1]
            return stop_session(path, state)
        if matching:
            path, state = matching[-1]
            if state["state"] in ("finalizing", "completed", "partial"):
                return stop_session(path, state)
        raise ValueError("active lecture archive not found")


class ControlHandler(socketserver.StreamRequestHandler):
    def handle(self):
        try:
            raw = self.rfile.readline(8193)
            if len(raw) > 8192 or not raw.endswith(b"\n"):
                raise ValueError("invalid control request")
            request = json.loads(raw)
            if not isinstance(request, dict):
                raise ValueError("invalid control request")
            result = {"ok": True, **control(self.server.cfg, request)}
        except (OSError, ValueError, RuntimeError, KeyError) as error:
            result = {"ok": False, "error": redact(str(error))[0]}
        self.wfile.write((json.dumps(result) + "\n").encode())


class ControlServer(socketserver.ThreadingUnixStreamServer):
    daemon_threads = True


def open_control_server(cfg):
    path = cfg["control_socket"]
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    if path.exists():
        if not path.is_socket():
            raise RuntimeError("control socket path is occupied")
        try:
            with socket.socket(socket.AF_UNIX) as probe:
                probe.connect(str(path))
        except ConnectionRefusedError:
            path.unlink()
        else:
            raise RuntimeError("control socket is already serving")
    server = ControlServer(str(path), ControlHandler)
    server.cfg = cfg
    os.chmod(path, 0o600)
    return server


def recover(path, state):
    if state["state"] in ("completed", "partial"):
        stop(path, state)
        return
    if not state["ended_at"]:
        state["ended_at"] = stamp()
        state["capture_to"] = stamp()
        mark_gap(path, state, "operator recovered a session without a recorded stop time")
    state["state"] = "interrupted"
    save_state(path, state)
    print(json.dumps({"session_id": state["session_id"], "state": state["state"], "path": str(path)}))


def service_once(cfg):
    for path in sessions(cfg["root"]):
        state = read_state(path)
        if state["state"] in ("completed", "partial"):
            if has_internal(path):
                cleanup_internal(path)
            continue
        if state["state"] not in ("active", "finalizing", "interrupted"):
            continue
        try:
            if shutil.disk_usage(cfg["root"]).free < cfg["min_free_bytes"]:
                mark_gap(path, state, "free space fell below collector reserve")
                state["state"] = "interrupted"
                save_state(path, state)
                continue
            collect(path, state, cfg)
            if state.get("capture_to") and dt.datetime.now(UTC) >= parse_stamp(state["capture_to"]):
                finalize(path, read_state(path))
        except Exception as error:
            state = read_state(path)
            warning = redact(str(error))[0]
            new_warning = warning not in state["warnings"]
            if new_warning:
                mark_gap(path, state, warning)
            state["state"] = "interrupted"
            save_state(path, state)
            if new_warning:
                print(f"collector session {state['session_id']}: {warning}", file=sys.stderr)


def verify(path):
    manifest = path / "manifest.json"
    checksums = path / "checksums.sha256"
    if not manifest.exists() or not checksums.exists():
        raise RuntimeError("archive is not finalized")
    data = json.loads(manifest.read_text())
    if data["state"] not in ("completed", "partial"):
        raise RuntimeError("archive is not finalized")
    listed = set()
    for line in checksums.read_text().splitlines():
        expected, name = line.split("  ", 1)
        relative = Path(name)
        if relative.is_absolute() or ".." in relative.parts or name in listed:
            raise RuntimeError("invalid checksum path")
        listed.add(name)
        target = path / name
        if target.is_symlink() or not target.is_file() or sha256_file(target) != expected:
            raise RuntimeError("checksum mismatch: " + name)
    actual = {str(file.relative_to(path)) for file in path.rglob("*") if file.is_file() and file.name != "checksums.sha256"}
    if listed != actual:
        raise RuntimeError("checksum file list does not match archive contents")
    print(json.dumps({"verified": True, "state": data["state"], "path": str(path)}))


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", default=os.environ.get("LECTURE_LOGS_CONFIG", str(DEFAULT_CONFIG)))
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("start").add_argument("--lecture-id", required=True)
    sub.add_parser("status")
    for name in ("stop", "recover"):
        sub.add_parser(name).add_argument("--session-id", required=True)
    sub.add_parser("verify").add_argument("--path", required=True)
    sub.add_parser("serve").add_argument("--once", action="store_true")
    args = parser.parse_args()
    if args.command == "verify":
        verify(Path(args.path))
        return
    cfg = config(args.config)
    with locked(cfg["root"]):
        if args.command == "start":
            start(cfg, args.lecture_id)
        elif args.command == "status":
            result = []
            for path in sessions(cfg["root"]):
                state = read_state(path)
                if state["state"] in ("completed", "partial"):
                    count = state["record_counts"]["all"]
                    size = (path / "all.log").stat().st_size
                else:
                    with db(path) as conn:
                        count = conn.execute("SELECT count(*) FROM events").fetchone()[0]
                        size = conn.execute("SELECT coalesce(sum(length(cast(message AS blob))),0) FROM events").fetchone()[0]
                result.append({"session_id": state["session_id"], "lecture_id": state["lecture_id"],
                               "state": "finalizing" if (path / "state.done").exists() else state["state"],
                               "path": str(path), "records": count,
                               "message_bytes": size, "last_event_at": state.get("last_event_at"),
                               "known_gaps": len(state["known_gaps"]), "warnings": state["warnings"]})
            print(json.dumps(result, ensure_ascii=False, indent=2))
        elif args.command in ("stop", "recover"):
            path = find_session(cfg["root"], args.session_id)
            (stop if args.command == "stop" else recover)(path, read_state(path))
        elif args.command == "serve":
            service_once(cfg)
    if args.command == "serve" and not args.once:
        if "control_socket" not in cfg:
            raise ValueError("control_socket is required for the collector service")
        server = open_control_server(cfg)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            while True:
                time.sleep(cfg.get("poll_seconds", 2))
                with locked(cfg["root"]):
                    service_once(cfg)
        finally:
            server.shutdown()
            server.server_close()
            cfg["control_socket"].unlink(missing_ok=True)


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, RuntimeError) as error:
        print(redact(str(error))[0], file=sys.stderr)
        sys.exit(1)
