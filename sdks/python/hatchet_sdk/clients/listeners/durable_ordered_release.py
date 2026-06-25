from __future__ import annotations

from dataclasses import dataclass, field
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from hatchet_sdk.clients.listeners.durable_event_listener import (
        DurableTaskEventLogEntryResult,
        PendingCallback,
    )

# How long the ordered-release gate stays closed waiting for a woken
# continuation to park (register its next awaited entry) before being forced
# open with a warning.
DEFAULT_PARK_TIMEOUT_S = 5.0

# How long a hole in the satisfied-order sequence may persist (while later
# completions are held) before the invocation's waiters are failed with a
# non-determinism error.
DEFAULT_GAP_TIMEOUT_S = 60.0


@dataclass
class OrderedReleaseGate:
    """Serializes the release of ordered entry_completed responses for a single
    durable task invocation. Completions are released to user code in
    satisfied_order; after a release wakes a parked continuation, further
    releases are held until that continuation parks again (registers its next
    awaited entry), or the park timeout elapses."""

    held: dict[int, tuple[PendingCallback, DurableTaskEventLogEntryResult]] = field(
        default_factory=dict
    )

    #: highest satisfied order released so far
    released: int = 0

    #: continuations woken by a gated release which have not yet parked; the
    #: gate is open iff wakes == 0
    wakes: int = 0

    #: when wakes last transitioned from zero (monotonic), for the park timeout
    wake_since: float = 0.0

    #: when the gate first became blocked on a missing order (monotonic), or
    #: None when not blocked
    gap_since: float | None = None
