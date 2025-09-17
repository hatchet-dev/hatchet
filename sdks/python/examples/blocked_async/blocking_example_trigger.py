# > Trigger
import time

from examples.blocked_async.blocking_example_worker import (
    blocking,
    non_blocking_async,
    non_blocking_sync,
)

non_blocking_sync.run_no_wait()
non_blocking_async.run_no_wait()

time.sleep(1)

blocking.run_no_wait()

time.sleep(1)

non_blocking_sync.run_no_wait()

# !!
