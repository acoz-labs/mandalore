"""Hermes owns the agent; Mandalore owns only portable semantic memory."""

import json
from pathlib import Path
import threading
import uuid

from .connection import Connection
from .transport import TransportError

UNAVAILABLE = "Mandalore memory is unavailable. Ask the Armorer to inspect the selected Hermes connection. Inspect local durability and delivery receipts before retrying."
CHECKPOINT = (
    "Hermes native memory remains independent and enabled as configured. "
    "At meaningful task checkpoints, use memory_remember for confirmed portable preferences, "
    "decisions and conventions, and memory_journal_append for useful semantic outcomes. "
    "Do not mirror native memory files, harvest transcripts, save secrets, or create a second extraction agent. "
    "Honor do-not-remember and no-journal requests. Preserve unresolved heads; never resurrect withdrawn "
    "knowledge from native notes. Reconcile stale native notes with current evidence and user direction. "
    "Do not copy signet records into native memory automatically. "
    "Disabling the native Mandalore plugin and starting a fresh session stops its tools and context; "
    "already loaded context cannot be revoked."
)


def attach(ctx, connection, active_home):
    """Registration is profile-scoped by Hermes; no process-global session state."""
    home = Path(connection.config["native_home"]).resolve()
    sessions = {}
    session_lock = threading.Lock()
    guidance = ("This Mandalore connection is enforced read-only. Do not save, journal or synchronize."
                if connection.config["read_only"] else CHECKPOINT)
    guidance += (" For memory work, use Hermes skill_view to read mandalore:this-is-the-way; "
                 "for requested connection administration, read mandalore:the-armorer.")
    ctx.register_system_prompt_section("mandalore.memory", guidance, position="after_memory")
    for name, description in (
        ("this-is-the-way", "Recall and save portable semantic knowledge through the selected signet."),
        ("the-armorer", "Inspect, install, update or repair an explicitly selected Mandalore connection."),
    ):
        ctx.register_skill(name, Path(__file__).resolve().parent / "skills" / name / "SKILL.md", description=description)

    def selected():
        if Path(active_home()).resolve() != home:
            raise TransportError("connection.invalid", "Mandalore belongs to a different Hermes profile.")

    def handler_for(name):
        def handler(args, **kwargs):
            try:
                selected()
                result = connection.call(name, args)
            except TransportError as error:
                result = error.envelope
            return json.dumps(result, ensure_ascii=False)
        return handler

    for op in connection.operations.values():
        handle = ctx.register_tool(
            name=op["name"], toolset="mandalore",
            schema={"name": op["name"], "description": op["description"], "parameters": op["input_schema"]},
            handler=handler_for(op["name"]), description=op["description"],
        )
        if handle is None:
            raise RuntimeError("Mandalore tool collision; no existing tool was replaced.")

    def before_turn(**kwargs):
        try:
            selected()
            session = kwargs.get("session_id")
            prompt = kwargs.get("user_message")
            turn = kwargs.get("turn_id") or str(uuid.uuid4())
            if (not isinstance(session, str) or not session or len(session.encode("utf-8")) > 256
                    or any(ord(ch) < 32 or ord(ch) == 127 for ch in session)):
                raise TransportError("session.invalid", "Hermes did not provide a bounded native session identity.")
            # Images/files still get refresh and orientation. Do not inspect their
            # contents or conversation history to construct a retrieval query.
            if not isinstance(prompt, str):
                prompt = ""
            with session_lock:
                if session not in sessions:
                    if len(sessions) >= 1024:
                        raise TransportError("session.limit", "Mandalore session inventory is full; start a fresh Hermes process.")
                    sessions[session] = {"lock": threading.Lock(), "started": False}
                state = sessions[session]
            if not state["lock"].acquire(blocking=False):
                raise TransportError("session.busy", "A Mandalore boundary for this session is already running.")
            try:
                kind = "turn" if state["started"] else ("startup" if kwargs.get("is_first_turn") else "resume")
                packet = connection.context(prompt, {"kind": kind, "session_id": session, "event_key": turn})
                state["started"] = True
            finally:
                state["lock"].release()
            with session_lock:
                if sessions.get(session) is not state:
                    raise TransportError("session.closed", "Hermes finalized or reset this session during refresh.")
            parts = [packet.get("context", ""), packet.get("warning", "")]
            return {"context": "\n\n".join(part for part in parts if part)}
        except TransportError:
            return {"context": UNAVAILABLE}

    ctx.register_hook("pre_llm_call", before_turn)

    def finalize(**kwargs):
        with session_lock:
            sessions.pop(kwargs.get("session_id"), None)

    def reset(**kwargs):
        with session_lock:
            sessions.pop(kwargs.get("old_session_id"), None)

    def before_command(**kwargs):
        # The CLI can restore a cached conversation without a start/finalize
        # event. Its command observer carries the departing native session ID.
        # Forget only that session; never infer a global active conversation.
        # A cancelled/failed switch conservatively refreshes on its next turn.
        if kwargs.get("surface") == "cli" and kwargs.get("command") in {"resume", "sessions", "branch", "new"}:
            session = kwargs.get("session_key")
            if (not isinstance(session, str) or not session or len(session.encode("utf-8")) > 256
                    or any(ord(ch) < 32 or ord(ch) == 127 for ch in session)):
                return
            with session_lock:
                sessions.pop(session, None)

    ctx.register_hook("on_session_finalize", finalize)
    ctx.register_hook("on_session_reset", reset)
    ctx.register_hook("pre_command", before_command)


def register(ctx):
    from hermes_constants import get_hermes_home
    connection = Connection(Path(__file__).resolve().parent, get_hermes_home())
    attach(ctx, connection, get_hermes_home)
