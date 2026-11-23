import time

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()

WORKER_NAME = "pause-on-sigterm-worker"


@hatchet.task()
def long_sleep(input: None, ctx: Context) -> None:
    time.sleep(6)


def main() -> None:
    worker = hatchet.worker(
        WORKER_NAME,
        workflows=[long_sleep],
    )
    worker.start()


if __name__ == "__main__":
    main()
