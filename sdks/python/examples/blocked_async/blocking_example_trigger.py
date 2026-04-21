# > Trigger
import time

from examples.blocked_async.blocking_example_worker import (
    blocking,
    non_blocking_async,
    non_blocking_sync,
)

non_blocking_sync.run(wait_for_result=False)
non_blocking_async.run(wait_for_result=False)

time.sleep(1)

blocking.run(wait_for_result=False)

time.sleep(1)

non_blocking_sync.run(wait_for_result=False)

# !!
