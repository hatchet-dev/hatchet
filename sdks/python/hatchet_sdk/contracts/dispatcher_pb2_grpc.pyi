"""Typing stubs for the generated gRPC module.

The generated `dispatcher_pb2_grpc.py` file does not include type annotations,
which causes mypy `no-untyped-call` errors when constructing `DispatcherStub`.

We keep this stub intentionally permissive (mostly `Any`) to avoid having to
track the full generated surface area while still satisfying strict mypy
settings for call sites.
"""

from __future__ import annotations

from typing import Any


class DispatcherStub:
    def __init__(self, channel: Any) -> None: ...

    # The generated stub attaches per-RPC callables as attributes. We type them
    # as `Any` to avoid depending on grpc's (sync vs aio) callable generics here.
    Register: Any
    Listen: Any
    ListenV2: Any
    Heartbeat: Any
    SubscribeToWorkflowEvents: Any
    SubscribeToWorkflowRuns: Any
    SendStepActionEvent: Any
    SendGroupKeyActionEvent: Any
    PutOverridesData: Any
    Unsubscribe: Any
    RefreshTimeout: Any
    ReleaseSlot: Any
    UpsertWorkerLabels: Any
    GetAction: Any
    GetActionV2: Any
    ReleaseAction: Any
    GetGroupKeyActions: Any
    ReleaseGroupKeyActions: Any
    ListActions: Any
