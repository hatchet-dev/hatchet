from datetime import timedelta

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()

WORKER_NAME = "heartbeat-delay-worker"

MAX_HEARTBEAT_DELAY_BEFORE_DECLARED_DEAD = timedelta(minutes=1)


@hatchet.task()
async def heartbeat_delay_task(input: None, ctx: Context) -> None:
    return None


def main() -> None:
    # > Configuring the max heartbeat delay
    worker = hatchet.worker(
        name=WORKER_NAME,
        workflows=[heartbeat_delay_task],
        max_heartbeat_delay_before_declared_dead=MAX_HEARTBEAT_DELAY_BEFORE_DECLARED_DEAD,
    )

    worker.start()


if __name__ == "__main__":
    main()
