"""Pinned native connection. The Go engine owns storage, schemas and delivery."""

import base64
import hashlib
import json
import os
from pathlib import Path
import re
import stat

from .transport import MAX_CATALOG, TransportError, run_json


def invalid():
    return TransportError("connection.invalid", "Mandalore connection changed or is invalid; inspect it with the Armorer.")


def read_bytes(path, limit):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, "rb") as source:
        info = os.fstat(source.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_size > limit:
            raise invalid()
        value = source.read(limit + 1)
        if len(value) > limit:
            raise invalid()
        return value


def digest(path, limit):
    return hashlib.sha256(read_bytes(path, limit)).hexdigest()


def package_digest(root):
    content = {}
    total = 0
    entries = 0
    for directory, dirs, files in os.walk(root, followlinks=False):
        if len(Path(directory).relative_to(root).parts) > 8:
            raise invalid()
        for name in dirs:
            if (Path(directory) / name).is_symlink():
                raise invalid()
        # Ignore real import caches, but reject redirected cache directories just
        # as the retained-generation verifier does.
        dirs[:] = [name for name in dirs if name != "__pycache__"]
        entries += len(dirs) + len(files)
        if entries > 256:
            raise invalid()
        for name in files:
            path = Path(directory) / name
            relative = path.relative_to(root).as_posix()
            if relative in ("connection.json", "session-policy.json"):
                continue
            raw = read_bytes(path, 1048576 - total)
            total += len(raw)
            content[relative] = base64.b64encode(raw).decode("ascii")
            if len(content) > 128:
                raise invalid()
    # Same sorted filename -> []byte encoding as Go's json.Marshal.
    raw = json.dumps(content, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    for char in "<>&\u2028\u2029":
        raw = raw.replace(char, "\\u%04x" % ord(char))
    return hashlib.sha256(raw.encode("utf-8")).hexdigest()


class Connection:
    def __init__(self, root, native_home):
        self.root = Path(root).resolve()
        try:
            c = json.loads(read_bytes(self.root / "connection.json", 16384))
            paths = {"runtime", "binding", "native_home", "native_binary", "state_dir", "connection_root"}
            keys = paths | {"schema_version", "harness", "runtime_sha256", "binding_sha256",
                            "signet_id", "package_sha256", "package_version", "read_only"}
            self.enabled = "session_policy" in c or "session_policy_sha256" in c
            if self.enabled:
                keys |= {"session_policy", "session_policy_sha256"}
            if (set(c) != keys or c["schema_version"] != 1 or c["harness"] != "hermes"
                    or type(c["read_only"]) is not bool
                    or not re.fullmatch(r"[a-z][a-z0-9-]{2,127}", c["signet_id"])
                    or not isinstance(c["package_version"], str) or not 0 < len(c["package_version"]) <= 128):
                raise invalid()
            for key in paths:
                if (not isinstance(c[key], str) or not os.path.isabs(c[key])
                        or any(ord(ch) < 32 or ord(ch) == 127 for ch in c[key])):
                    raise invalid()
            for key in ("runtime_sha256", "binding_sha256", "package_sha256"):
                if not re.fullmatch(r"[a-f0-9]{64}", c[key]):
                    raise invalid()
            if (self.root != (Path(c["connection_root"]) / "package").resolve()
                    or Path(native_home).resolve() != Path(c["native_home"]).resolve()
                    or package_digest(self.root) != c["package_sha256"]):
                raise invalid()
            if self.enabled and (c["read_only"] or c["session_policy"] != str(Path(c["connection_root"]) / "package/session-policy.json")
                                 or digest(c["session_policy"], 16384) != c["session_policy_sha256"]):
                raise invalid()
            self.config = c
            self.stamp = self.runtime_stamp()
            if digest(c["runtime"], 134217728) != c["runtime_sha256"] or self.runtime_stamp() != self.stamp:
                raise invalid()
            self.guards = ["--binding", c["binding"], "--binding-sha256", c["binding_sha256"],
                           "--signet-id", c["signet_id"], "--harness", "hermes"]
            if self.enabled:
                self.guards += ["--session-policy", c["session_policy"], "--session-policy-sha256", c["session_policy_sha256"]]
            if c["read_only"]:
                self.guards += ["--read-only"]
            info = self.require_ok(self.request(["call", "hermes_package_inspect"], {}, timeout=5))
            if (info["name"] != "mandalore" or info["harness_protocol_version"] != 1
                    or info["sha256"] != c["package_sha256"] or info["version"] != c["package_version"]):
                raise invalid()
            if self.enabled:
                catalog = self.require_ok(self.bound("memory_session_catalog", {}, timeout=5, max_output=MAX_CATALOG))
            else:
                catalog = self.require_ok(self.request(["operations"], timeout=5, max_output=MAX_CATALOG))
            operations = [op for op in catalog["operations"] if op.get("requires_binding") is True and op.get("cli_only") is False]
            self.operations = {op["name"]: op for op in operations}
            if not operations or len(self.operations) != len(operations):
                raise invalid()
            for op in operations:
                if (not re.fullmatch(r"[a-z][a-z0-9_]{0,63}", op["name"])
                        or not isinstance(op["description"], str) or type(op["read_only"]) is not bool
                        or not isinstance(op["input_schema"], dict)):
                    raise invalid()
        except (OSError, ValueError, TypeError, KeyError):
            raise invalid() from None

    def runtime_stamp(self):
        info = os.lstat(self.config["runtime"])
        if not stat.S_ISREG(info.st_mode) or not info.st_mode & 0o111 or not 0 < info.st_size <= 134217728:
            raise invalid()
        return (info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns, info.st_ctime_ns)

    def request(self, args, value=None, **options):
        try:
            if self.runtime_stamp() != self.stamp:
                raise invalid()
        except OSError:
            raise invalid() from None
        return run_json(self.config["runtime"], args, value, **options)

    def bound(self, name, value, **options):
        return self.request(["call", name, *self.guards], value, **options)

    @staticmethod
    def require_ok(envelope):
        if not envelope["ok"]:
            raise invalid()
        return envelope["result"]

    def call(self, name, value):
        if name not in self.operations:
            raise TransportError("operation.unknown", "Operation is not a native memory tool.")
        return self.bound(name, value, mutating=not self.operations[name]["read_only"])

    def context(self, prompt, boundary):
        value = {"prompt": prompt.encode("utf-8")[:2048].decode("utf-8", errors="ignore")}
        if self.enabled:
            value["boundary"] = boundary
        packet = self.require_ok(self.bound("memory_context", value, timeout=9 if self.enabled else 5, mutating=self.enabled))
        if (not isinstance(packet, dict) or set(packet) - {"context", "warning", "synchronization"}
                or any(not isinstance(packet[key], str) for key in ("context", "warning") if key in packet)
                or not (packet.get("context") or packet.get("warning"))
                or len(json.dumps(packet).encode("utf-8")) > 16383):
            raise invalid()
        return packet
