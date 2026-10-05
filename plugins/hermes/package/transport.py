"""Bounded, shell-free calls to the shared engine; no retries or memory logic."""

import json
import os
import selectors
import signal
import subprocess
import time

MAX_INPUT = 32768
MAX_OUTPUT = 65536
MAX_CATALOG = 1048576


class TransportError(Exception):
    def __init__(self, code, message, may_write=False):
        super().__init__(message)
        self.envelope = {
            "protocol_version": 1, "ok": False,
            "error": {"code": code, "message": message, "retryable": False,
                      "write_may_have_occurred": may_write,
                      "inspect_before_retry": may_write},
        }


def decode(raw, exit_code):
    value = json.loads(raw.decode("utf-8"))
    if (not isinstance(value, dict) or value.get("protocol_version") != 1
            or type(value.get("ok")) is not bool
            or set(value) - {"protocol_version", "ok", "result", "error", "session_sync"}):
        raise ValueError("Invalid envelope")
    if "session_sync" in value and not isinstance(value["session_sync"], dict):
        raise ValueError("Invalid receipt")
    if value["ok"]:
        if exit_code != 0 or "result" not in value or "error" in value:
            raise ValueError("Contradictory success")
    elif (exit_code == 0 or not isinstance(value.get("error"), dict)
          or not isinstance(value["error"].get("code"), str)
          or not isinstance(value["error"].get("message"), str) or "result" in value):
        raise ValueError("Contradictory failure")
    return value


def run_json(executable, args, value=None, *, timeout=45, max_output=MAX_OUTPUT, mutating=False):
    if not 0 < timeout <= 45 or not 0 < max_output <= MAX_CATALOG:
        raise TransportError("input.invalid", "Invalid Mandalore transport limits.")
    try:
        payload = b"" if value is None else json.dumps(value, allow_nan=False).encode("utf-8")
        if len(payload) > MAX_INPUT:
            raise ValueError()
    except (ValueError, TypeError, UnicodeError):
        raise TransportError("input.invalid", "Mandalore input must be bounded JSON.") from None
    try:
        child = subprocess.Popen([executable, *args], stdin=subprocess.PIPE,
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                 start_new_session=True)
    except OSError:
        raise TransportError("operation.io", "Mandalore executable could not be started.") from None

    output = bytearray()
    diagnostics = 0
    sent = 0
    fault = None
    forced = False
    deadline = time.monotonic() + timeout

    def kill(sig):
        try:
            os.killpg(child.pid, sig)
        except ProcessLookupError:
            pass

    try:
        with selectors.DefaultSelector() as poll:
            for stream in (child.stdout, child.stderr, child.stdin):
                os.set_blocking(stream.fileno(), False)
            poll.register(child.stdout, selectors.EVENT_READ)
            poll.register(child.stderr, selectors.EVENT_READ)
            if payload:
                poll.register(child.stdin, selectors.EVENT_WRITE)
            else:
                child.stdin.close()
            while poll.get_map() or child.poll() is None:
                if time.monotonic() >= deadline:
                    if fault is None:
                        fault = "operation.cancelled"
                        kill(signal.SIGTERM)
                        deadline = time.monotonic() + 1
                    else:
                        forced = True
                        kill(signal.SIGKILL)
                        break
                for key, _ in poll.select(max(0, min(0.1, deadline - time.monotonic()))):
                    stream = key.fileobj
                    if stream is child.stdin:
                        try:
                            sent += os.write(stream.fileno(), payload[sent:sent + 4096])
                        except BrokenPipeError:
                            sent = len(payload)
                        if sent == len(payload):
                            poll.unregister(stream)
                            stream.close()
                        continue
                    data = os.read(stream.fileno(), 65536)
                    if not data:
                        poll.unregister(stream)
                        continue
                    if stream is child.stdout:
                        if len(output) <= max_output:
                            output.extend(data)
                    else:
                        diagnostics += len(data)
                    if len(output) > max_output or diagnostics > MAX_OUTPUT:
                        fault = "output.invalid"
                        forced = True
                        kill(signal.SIGKILL)
                        break
                if forced:
                    break
        child.wait(timeout=1)
        if not forced:
            try:
                return decode(output, child.returncode)
            except (ValueError, UnicodeError):
                pass
        raise TransportError(fault or "output.invalid",
                             "Mandalore did not return a complete valid receipt; inspect possible effects before retrying.",
                             mutating)
    finally:
        # Also covers interrupted Python calls. Never leave a sync subprocess
        # running after Hermes has abandoned its foreground operation.
        kill(signal.SIGKILL)
        child.wait()
        for stream in (child.stdin, child.stdout, child.stderr):
            stream.close()
