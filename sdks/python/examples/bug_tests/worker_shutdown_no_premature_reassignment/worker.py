import os
import time
from datetime import timedelta

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()

TASK_NAME = "shutdown-drain-task"

# Must clear the engine's hardcoded 30s heartbeat-staleness threshold for
# reassignment (see pkg/repository/sqlcv1/tasks.sql: ListTasksToReassign) with
# enough margin for the reassignment poller to have run at least once, so that
# this test can actually distinguish "never reassigned" from "reassignment
# just hasn't happened yet".
SLEEP_SECONDS = 40


@hatchet.task(name=TASK_NAME, execution_timeout=timedelta(seconds=60))
def drain_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    log_path = os.environ["SHUTDOWN_TEST_LOG_PATH"]
    worker_name = os.environ["HATCHET_TEST_WORKER_NAME"]

    with open(log_path, "a") as f:
        f.write(f"{worker_name} START\n")

    time.sleep(SLEEP_SECONDS)

    with open(log_path, "a") as f:
        f.write(f"{worker_name} FINISH\n")

    return {"worker": worker_name}


def main() -> None:
    worker_name = os.environ["HATCHET_TEST_WORKER_NAME"]
    worker = hatchet.worker(worker_name, workflows=[drain_task])
    worker.start()


if __name__ == "__main__":
    main()
