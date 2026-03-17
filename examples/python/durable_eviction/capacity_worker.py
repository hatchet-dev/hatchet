"""
Dedicated worker for capacity-eviction e2e tests.

Runs with durable_slots=1 so that a single waiting durable task triggers
capacity pressure and gets evicted (even with ttl=None).
"""

from __future__ import annotations

from examples.durable_eviction.worker import capacity_evictable_sleep, hatchet


def main() -> None:
    worker = hatchet.worker(
        "capacity-eviction-worker",
        durable_slots=1,
        workflows=[capacity_evictable_sleep],
    )
    worker.start()


if __name__ == "__main__":
    main()
