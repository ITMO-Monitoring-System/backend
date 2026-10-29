import contextlib
import importlib.util
import io
import json
import hashlib
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from types import SimpleNamespace
from pathlib import Path
from unittest.mock import patch


MODULE = Path(__file__).resolve().parents[1] / "lecture_logs.py"
spec = importlib.util.spec_from_file_location("lecture_logs", MODULE)
ll = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ll)


class LectureLogsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root = Path(self.temp.name)
        self.cfg = {"root": root / "archives", "pre_roll_seconds": 120, "poll_seconds": 1,
                    "post_roll_seconds": 0, "min_free_bytes": 0, "require_service": False,
                    "require_containers": False}
        self.cfg["root"].mkdir()
        self.journal = root / "journal.json"
        self.fake = root / "journalctl"
        self.fake.write_text("""#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
events = json.loads(Path(os.environ['TEST_JOURNAL']).read_text())
args = sys.argv[1:]
if '--cursor' in args:
    cursor = args[args.index('--cursor') + 1]
    events = [e for e in events if e.get('__CURSOR') == cursor]
elif '--after-cursor' in args:
    cursor = args[args.index('--after-cursor') + 1]
    try:
        index = next(i for i, e in enumerate(events) if e.get('__CURSOR') == cursor)
        events = events[index + 1:]
    except StopIteration:
        events = []
if '--since' in args:
    start = int(args[args.index('--since') + 1][1:]) * 1000000
    events = [e for e in events if int(e['__REALTIME_TIMESTAMP']) >= start]
if '--until' in args:
    end = int(args[args.index('--until') + 1][1:]) * 1000000
    events = [e for e in events if int(e['__REALTIME_TIMESTAMP']) <= end]
if '-n' in args:
    limit = args[args.index('-n') + 1]
    events = events[:int(limit[1:])] if limit.startswith('+') else events[-int(limit):]
for event in events:
    print(json.dumps(event))
""")
        self.fake.chmod(0o700)
        self.env = patch.dict(os.environ, {"LECTURE_LOGS_JOURNALCTL": str(self.fake),
                                        "TEST_JOURNAL": str(self.journal)})
        self.env.start()
        self.addCleanup(self.env.stop)
        now = time.time_ns() // 1000
        self.events = [
            self.entry("c1", now - 1000, "backend", json.dumps({"event": "visit_written", "lecture_id": 12,
                "request_id": "r1", "message": "amqp://user:secret@host"})),
            self.entry("c2", now, "face_tracking", '{"event":"crop_published","lecture_id":123,"request_id":"r2"}'),
            self.entry("c3", now, "frontend", "random 12 text"),
            {"__CURSOR": "c4", "__REALTIME_TIMESTAMP": str(now), "_SYSTEMD_UNIT": "docker.service",
             "MESSAGE": "Authorization: Bearer secret-token"},
            self.entry("c5", now, "backend", "password=foo\nrtsp://user:camera-pass@host/live\ntraceback line 2" + "x" * 50000),
            {"__CURSOR": "c6", "__REALTIME_TIMESTAMP": str(now), "CONTAINER_TAG": "unrelated/backend",
             "MESSAGE": "foreign"},
            self.entry("c7", now, "face_recognizing", json.dumps({"event": "unsafe_legacy", "lecture_id": 12,
                "image_b64": "somebase64", "Authorization": "Bearer secret-token"})),
            self.entry("c8", now, "frontend", 'upstream failed, request: "GET /api/users/123?token=private HTTP/1.1"'),
        ]
        self.journal.write_text(json.dumps(self.events))

    def entry(self, cursor, ts, service, message):
        return {"__CURSOR": cursor, "__REALTIME_TIMESTAMP": str(ts), "_BOOT_ID": "boot-1",
                "CONTAINER_TAG": "itmo-monitoring/" + service, "CONTAINER_ID_FULL": cursor + "-container",
                "IMAGE_NAME": "example/" + service, "MESSAGE": message}

    def begin(self, lecture):
        with contextlib.redirect_stdout(io.StringIO()) as output:
            ll.start(self.cfg, str(lecture))
        result = json.loads(output.getvalue())
        return Path(result["path"]), result["session_id"]

    def test_export_exact_id_redaction_and_checksums(self):
        path, _ = self.begin(12)
        state = ll.read_state(path)
        ll.collect(path, state, self.cfg)
        ll.collect(path, ll.read_state(path), self.cfg)
        with contextlib.redirect_stdout(io.StringIO()):
            ll.stop(path, ll.read_state(path))
        ll.service_once(self.cfg)
        data = json.loads((path / "manifest.json").read_text())
        self.assertEqual(data["state"], "completed")
        self.assertEqual(data["record_counts"]["all"], 7)
        self.assertEqual(data["record_counts"]["lecture_only"], 2)
        self.assertEqual(len((path / "services/backend.log").read_text().splitlines()), 2)
        contents = (path / "all.log").read_text()
        self.assertNotIn("secret", contents)
        self.assertNotIn("password=foo", contents)
        self.assertNotIn("camera-pass", contents)
        self.assertNotIn("foreign", contents)
        self.assertNotIn("somebase64", contents)
        self.assertNotIn("private", contents)
        self.assertNotIn("/api/users/123", contents)
        self.assertIn("x" * 50000, contents)
        self.assertEqual(len(contents.splitlines()), 7)
        self.assertIn("c1-container", data["containers_seen"])
        self.assertIn("c5-container", data["containers_seen"])
        self.assertFalse((path / "events.sqlite3").exists())
        self.assertFalse((path / "state.json").exists())
        self.assertEqual(path.stat().st_mode & 0o777, 0o700)
        self.assertEqual((path / "all.log").stat().st_mode & 0o777, 0o600)
        with contextlib.redirect_stdout(io.StringIO()):
            ll.verify(path)
        (path / "state.done").write_text("pending cleanup")
        (path / "events.sqlite3").write_text("stale buffer")
        ll.service_once(self.cfg)
        self.assertFalse((path / "state.done").exists())
        self.assertEqual((path / "all.log").read_text(), contents)
        with (path / "all.log").open("a") as file:
            file.write("tampered")
        with self.assertRaisesRegex(RuntimeError, "checksum mismatch"):
            ll.verify(path)

    def test_parallel_sessions_and_missing_cursor(self):
        path1, _ = self.begin(12)
        path2, _ = self.begin(123)
        ll.collect(path1, ll.read_state(path1), self.cfg)
        ll.collect(path2, ll.read_state(path2), self.cfg)
        self.assertNotEqual(path1, path2)
        self.assertIn(ll.read_state(path1)["session_id"], ll.read_state(path2)["overlaps_with"])
        self.journal.write_text(json.dumps(self.events[2:5]))
        ll.collect(path1, ll.read_state(path1), self.cfg)
        self.assertTrue(ll.read_state(path1)["known_gaps"])
        with contextlib.redirect_stdout(io.StringIO()):
            ll.stop(path1, ll.read_state(path1))
        ll.service_once(self.cfg)
        self.assertEqual(ll.read_state(path1)["state"], "partial")
        self.assertEqual(ll.read_state(path2)["state"], "active")

    def test_recover_marks_interruption(self):
        path, _ = self.begin(12)
        with contextlib.redirect_stdout(io.StringIO()):
            ll.recover(path, ll.read_state(path))
        ll.service_once(self.cfg)
        self.assertEqual(ll.read_state(path)["state"], "partial")

    def test_low_space_is_visible(self):
        path, _ = self.begin(12)
        self.cfg["min_free_bytes"] = 1
        with patch.object(ll.shutil, "disk_usage", return_value=SimpleNamespace(free=0)):
            ll.service_once(self.cfg)
        state = ll.read_state(path)
        self.assertEqual(state["state"], "interrupted")
        self.assertIn("free space", state["known_gaps"][0]["reason"])

    def test_command_interface_survives_process_restarts(self):
        installed_script = Path(self.temp.name) / "lecture_logs.py"
        shutil.copy2(MODULE, installed_script)
        config_path = installed_script.with_name("lecture-logs.json")
        config_path.write_text(json.dumps({**self.cfg, "root": str(self.cfg["root"])}))

        def command(*args):
            return subprocess.run([sys.executable, str(installed_script), *args],
                                  check=True, text=True, capture_output=True).stdout

        started = json.loads(command("start", "--lecture-id", "12"))
        self.assertEqual(json.loads(command("status"))[0]["state"], "active")
        command("serve", "--once")
        self.assertEqual(json.loads(command("status"))[0]["records"], 7)
        command("stop", "--session-id", started["session_id"])
        command("serve", "--once")
        self.assertEqual(json.loads(command("status"))[0]["state"], "completed")
        self.assertTrue(json.loads(command("verify", "--path", started["path"]))["verified"])

    def test_button_control_keeps_previous_archive(self):
        self.cfg["control_socket"] = Path(self.temp.name) / "control" / "collector.sock"
        server = ll.open_control_server(self.cfg)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(lambda: (server.shutdown(), server.server_close(), thread.join()))

        def command(action):
            with socket.socket(socket.AF_UNIX) as client:
                client.connect(str(self.cfg["control_socket"]))
                client.sendall((json.dumps({"action": action, "lecture_id": "12"}) + "\n").encode())
                with client.makefile("r") as stream:
                    return json.loads(stream.readline())

        first = command("start")
        self.assertTrue(first["ok"])
        self.assertEqual(command("start")["session_id"], first["session_id"])
        first_path = Path(first["path"])
        ll.service_once(self.cfg)
        self.assertEqual(command("stop")["session_id"], first["session_id"])
        ll.service_once(self.cfg)
        with contextlib.redirect_stdout(io.StringIO()):
            ll.verify(first_path)
        first_hash = hashlib.sha256((first_path / "all.log").read_bytes()).hexdigest()

        second = command("start")
        self.assertTrue(second["ok"])
        self.assertNotEqual(second["session_id"], first["session_id"])
        self.assertNotEqual(second["path"], first["path"])
        ll.service_once(self.cfg)
        self.assertEqual(command("stop")["session_id"], second["session_id"])
        ll.service_once(self.cfg)
        with contextlib.redirect_stdout(io.StringIO()):
            ll.verify(Path(second["path"]))
            ll.verify(first_path)
        self.assertEqual(hashlib.sha256((first_path / "all.log").read_bytes()).hexdigest(), first_hash)
        self.assertEqual(ll.read_state(first_path)["state"], "completed")


if __name__ == "__main__":
    unittest.main()
