"""Synthetic host contract tests; never open the operator's native profile."""

import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import threading
import unittest

ROOT = Path(__file__).resolve().parents[1] / "package"
spec = importlib.util.spec_from_file_location("mandalore_test_plugin", ROOT / "__init__.py",
                                             submodule_search_locations=[str(ROOT)])
plugin = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = plugin
spec.loader.exec_module(plugin)
transport = sys.modules[spec.name + ".transport"]
connection = sys.modules[spec.name + ".connection"]


class Host:
    def __init__(self):
        self.tools = {}
        self.hooks = {}
        self.sections = {}
        self.skills = {}

    def register_tool(self, **args):
        if args["name"] in self.tools:
            return None
        self.tools[args["name"]] = args
        return object()

    def register_hook(self, name, callback):
        self.hooks[name] = callback

    def register_system_prompt_section(self, name, content, **kwargs):
        self.sections[name] = content

    def register_skill(self, name, path, **kwargs):
        assert path.is_file()
        self.skills[name] = path


class FakeConnection:
    def __init__(self, home, readonly=False):
        self.config = {"native_home": home, "read_only": readonly}
        self.operations = {"memory_remember": {"name": "memory_remember", "description": "Save semantic knowledge.",
                                               "input_schema": {"type": "object"}, "read_only": False}}
        self.calls = []

    def call(self, name, value):
        self.calls.append((name, value))
        return {"protocol_version": 1, "ok": True, "result": {"durable_locally": True},
                "session_sync": {"attempted": True, "error": {"code": "session.cancelled"}}}

    def context(self, prompt, boundary):
        self.calls.append((prompt, boundary))
        return {"context": "Untrusted evidence", "warning": "Delivery pending"}


class AdapterTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.home = Path(self.temp.name)
        self.host = Host()
        self.connection = FakeConnection(str(self.home))
        plugin.attach(self.host, self.connection, lambda: self.home)

    def test_turn_refresh_precedes_context_and_uses_host_identity(self):
        result = self.host.hooks["pre_llm_call"](session_id="chat-a", turn_id="turn-b", user_message="Question",
                                                 is_first_turn=True,
                                                 conversation_history=[{"secret": "must not be exported"}])
        self.assertEqual(self.connection.calls, [("Question", {"kind": "startup", "session_id": "chat-a", "event_key": "turn-b"})])
        self.assertIn("Untrusted evidence", result["context"])
        self.assertIn("Delivery pending", result["context"])
        self.assertEqual(set(self.host.hooks), {"pre_llm_call", "on_session_finalize", "on_session_reset", "pre_command"})
        self.assertIn("Do not mirror", self.host.sections["mandalore.memory"])

    def test_resume_multimodal_and_finalize_boundaries(self):
        hook = self.host.hooks["pre_llm_call"]
        hook(session_id="resumed", user_message=[{"type": "image", "data": "not extracted"}], is_first_turn=False)
        self.assertEqual(self.connection.calls[-1][0], "")
        self.assertEqual(self.connection.calls[-1][1]["kind"], "resume")
        hook(session_id="resumed", user_message="next")
        self.assertEqual(self.connection.calls[-1][1]["kind"], "turn")
        self.host.hooks["on_session_finalize"](session_id="resumed")
        hook(session_id="resumed", user_message="new process resume")
        self.assertEqual(self.connection.calls[-1][1]["kind"], "resume")

    def test_missing_session_never_uses_another_conversation(self):
        result = self.host.hooks["pre_llm_call"](user_message="Question")
        self.assertIn("unavailable", result["context"])
        self.assertEqual(self.connection.calls, [])

    def test_cli_switch_back_refreshes_only_departing_session(self):
        turn = self.host.hooks["pre_llm_call"]
        command = self.host.hooks["pre_command"]
        turn(session_id="a", user_message="first", is_first_turn=True)
        turn(session_id="gateway", user_message="concurrent", is_first_turn=True)
        command(surface="cli", command="resume", session_key="a", args_raw="private title")
        turn(session_id="b", user_message="resumed", is_first_turn=False)
        command(surface="cli", command="resume", session_key="b")
        turn(session_id="a", user_message="returned", is_first_turn=False)
        self.assertEqual(self.connection.calls[-1][1]["kind"], "resume")
        turn(session_id="gateway", user_message="still running")
        self.assertEqual(self.connection.calls[-1][1]["kind"], "turn")
        command(surface="gateway", command="resume", session_key="gateway")
        command(surface="cli", command="help", session_key="gateway")
        turn(session_id="gateway", user_message="still running")
        self.assertEqual(self.connection.calls[-1][1]["kind"], "turn")

    def test_reset_during_refresh_discards_stale_context(self):
        entered, finish = threading.Event(), threading.Event()
        original = self.connection.context
        def blocked(prompt, boundary):
            entered.set()
            self.assertTrue(finish.wait(2))
            return original(prompt, boundary)
        self.connection.context = blocked
        replies = []
        worker = threading.Thread(target=lambda: replies.append(self.host.hooks["pre_llm_call"](session_id="old", user_message="Question")))
        worker.start()
        self.assertTrue(entered.wait(2))
        self.host.hooks["on_session_reset"](old_session_id="old", new_session_id="new")
        finish.set()
        worker.join(2)
        self.assertFalse(worker.is_alive())
        self.assertIn("unavailable", replies[0]["context"])
        self.assertNotIn("Untrusted evidence", replies[0]["context"])

    def test_profile_switch_blocks_tools_and_context(self):
        other = Host()
        plugin.attach(other, self.connection, lambda: self.home / "other")
        result = json.loads(other.tools["memory_remember"]["handler"]({"body": "example"}))
        self.assertFalse(result["ok"])
        other.hooks["pre_llm_call"](session_id="a", user_message="Question")
        self.assertEqual(self.connection.calls, [])

    def test_save_is_once_and_full_local_and_delivery_receipt_survives(self):
        raw = self.host.tools["memory_remember"]["handler"]({"body": "confirmed"}, session_id="a")
        result = json.loads(raw)
        self.assertTrue(result["result"]["durable_locally"])
        self.assertEqual(result["session_sync"]["error"]["code"], "session.cancelled")
        self.assertEqual(len(self.connection.calls), 1)

    def test_readonly_guidance_has_no_save_checkpoint(self):
        host = Host()
        plugin.attach(host, FakeConnection(str(self.home), True), lambda: self.home)
        self.assertIn("Do not save", host.sections["mandalore.memory"])
        self.assertNotIn("meaningful task checkpoints", host.sections["mandalore.memory"])

    def test_collision_does_not_replace_host_tool(self):
        original = self.host.tools["memory_remember"]
        with self.assertRaises(RuntimeError):
            plugin.attach(self.host, self.connection, lambda: self.home)
        self.assertIs(self.host.tools["memory_remember"], original)

    def test_package_hash_detects_source_changes_and_rejects_symlinks(self):
        package = self.home / "package"
        package.mkdir()
        (package / "a.py").write_text("first")
        before = connection.package_digest(package)
        (package / "connection.json").write_text("local")
        self.assertEqual(connection.package_digest(package), before)
        (package / "a.py").write_text("second")
        self.assertNotEqual(connection.package_digest(package), before)
        (package / "linked.py").symlink_to(package / "a.py")
        with self.assertRaises(OSError):
            connection.package_digest(package)

    def test_package_hash_ignores_real_cache_but_rejects_redirected_cache(self):
        package = self.home / "package"
        package.mkdir()
        (package / "a.py").write_text("source")
        before = connection.package_digest(package)
        cache = package / "__pycache__"
        cache.mkdir()
        (cache / "a.pyc").write_bytes(b"generated")
        self.assertEqual(connection.package_digest(package), before)
        (cache / "a.pyc").unlink()
        cache.rmdir()
        cache.symlink_to(self.home, target_is_directory=True)
        with self.assertRaises(transport.TransportError):
            connection.package_digest(package)


class TransportTests(unittest.TestCase):
    def run_python(self, script, value=None, **options):
        return transport.run_json(sys.executable, ["-c", script], value, **options)

    def test_success_and_complete_failure(self):
        result = self.run_python('print(\'{"protocol_version":1,"ok":true,"result":{"saved":true}}\')')
        self.assertTrue(result["result"]["saved"])
        result = self.run_python('import sys; print(\'{"protocol_version":1,"ok":false,"error":{"code":"store.busy","message":"busy"}}\'); sys.exit(3)')
        self.assertFalse(result["ok"])

    def test_contradictory_and_truncated_responses_preserve_uncertainty(self):
        for script in ('print(\'{"protocol_version":1,"ok":false,"error":{"code":"x","message":"x"}}\')', 'print("{")'):
            with self.assertRaises(transport.TransportError) as caught:
                self.run_python(script, mutating=True)
            self.assertTrue(caught.exception.envelope["error"]["write_may_have_occurred"])

    def test_input_limits_reject_before_execution(self):
        with self.assertRaises(transport.TransportError) as caught:
            self.run_python('raise Exception("must not run")', {"body": "x" * 40000}, mutating=True)
        self.assertFalse(caught.exception.envelope["error"]["write_may_have_occurred"])

    def test_output_and_diagnostic_flood_are_bounded(self):
        for target in ("sys.stdout", "sys.stderr"):
            with self.assertRaises(transport.TransportError) as caught:
                self.run_python(f'import sys; {target}.write("x" * 200000)', mutating=True)
            self.assertEqual(caught.exception.envelope["error"]["code"], "output.invalid")

    def test_deadline_terminates_process_group(self):
        with self.assertRaises(transport.TransportError) as caught:
            self.run_python('import time; time.sleep(60)', timeout=0.1, mutating=True)
        self.assertEqual(caught.exception.envelope["error"]["code"], "operation.cancelled")

    def test_complete_cancellation_receipt_survives_term(self):
        script = '''import signal,time,sys
def stop(*_):
 print('{"protocol_version":1,"ok":false,"error":{"code":"operation.cancelled","message":"interrupted"},"session_sync":{"attempted":true}}', flush=True)
 sys.exit(130)
signal.signal(signal.SIGTERM,stop)
time.sleep(60)
'''
        result = self.run_python(script, timeout=0.5, mutating=True)
        self.assertTrue(result["session_sync"]["attempted"])


if __name__ == "__main__":
    unittest.main()
