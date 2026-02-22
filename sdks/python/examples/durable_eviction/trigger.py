"""Quick manual trigger for the evictable_sleep task."""

from examples.durable_eviction.worker import evictable_sleep

ref = evictable_sleep.run_no_wait()
print(f"Triggered evictable_sleep: workflow_run_id={ref.workflow_run_id}")
