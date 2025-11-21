#!/usr/bin/env python3
"""Compression test script for Python SDK"""

import os
import signal
import threading
import time
from datetime import datetime
from typing import Any

from hatchet_sdk import Context, Hatchet

# Create large payload (100KB)
def create_large_payload() -> dict[str, str]:
    payload: dict[str, str] = {}
    chunk = "a" * 1000  # 1KB chunk
    for i in range(100):
        payload[f"chunk_{i}"] = chunk
    return payload


def emit_events(hatchet: Hatchet, total_events: int, events_per_second: int) -> None:
    """Emit events in a separate thread"""
    interval = 1.0 / events_per_second
    large_payload = create_large_payload()
    event_id = 0

    print(f"Starting to emit {total_events} events...")

    while event_id < total_events:
        event = {
            "id": event_id,
            "createdAt": datetime.now().isoformat(),
            "payload": large_payload,
        }

        try:
            hatchet.event.push("compression-test:event", event)
            event_id += 1
            if event_id % 50 == 0:
                print(f"Emitted {event_id} events...")
        except Exception as e:
            print(f"Error pushing event {event_id}: {e}")

        # Wait for next interval
        time.sleep(interval)

    print(f"Finished emitting {event_id} events")


def main() -> None:
    # Namespace is set via environment variable HATCHET_CLIENT_NAMESPACE
    hatchet = Hatchet(debug=False)

    # Get compression state from environment (default to 'enabled')
    compression_state = os.getenv("COMPRESSION_STATE", "enabled")
    workflow_name = f"{compression_state}-python"

    # Create workflow
    workflow = hatchet.workflow(
        name=workflow_name,
        on_events=["compression-test:event"],
    )

    @workflow.task()
    def step1(input_data: Any, ctx: Context) -> dict[str, Any]:
        # EmptyModel allows extra fields, access as attributes
        # The event data is passed as the workflow input
        event_id = getattr(input_data, "id", None)
        print(f"Processing event {event_id}")
        return {
            "processed": True,
            "eventId": event_id,
            "timestamp": datetime.now().isoformat(),
        }

    # Create worker
    worker = hatchet.worker(
        "compression-test-worker",
        slots=100,
        workflows=[workflow],
    )

    # Get number of events from environment variable
    total_events = int(os.getenv("TEST_EVENTS_COUNT", "10"))
    events_per_second = 10
    # Calculate duration needed to send all events
    duration = max(1, total_events / events_per_second)  # At least 1 second

    # Calculate total wait time (worker registration + event emission + processing buffer)
    wait_time = int(duration) + 15  # 5s for registration + duration + 10s buffer

    # Set up signal handler to stop worker after test duration
    def stop_worker_after_delay():
        time.sleep(wait_time)
        print("Test complete, stopping worker...")
        # Send SIGTERM to current process to trigger worker shutdown
        os.kill(os.getpid(), signal.SIGTERM)

    # Start timer to stop worker after test duration
    stop_timer = threading.Timer(wait_time, lambda: os.kill(os.getpid(), signal.SIGTERM))
    stop_timer.daemon = True
    stop_timer.start()

    # Start emitting events in a separate thread
    emit_thread = threading.Thread(
        target=emit_events,
        args=(hatchet, total_events, events_per_second),
        daemon=True,
    )

    print("Starting worker...")
    print(f"Emitting {total_events} events over {duration:.1f} seconds...")

    # Start emitting events
    emit_thread.start()

    # Wait a moment for events to start
    time.sleep(1)

    # Start worker (blocking call - will run until SIGTERM)
    try:
        worker.start()
    except KeyboardInterrupt:
        print("Worker stopped")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("Test interrupted")
    except Exception as e:
        print(f"Test failed: {e}")
        raise
