from examples.durable_eviction.worker import evictable_sleep

ref = evictable_sleep.run(wait_for_result=False)
print(f"Triggered evictable_sleep: workflow_run_id={ref.workflow_run_id}")
