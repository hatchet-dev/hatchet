from datetime import timedelta
import ctypes

from hatchet_sdk import Context, Hatchet

import argparse

from hatchet_sdk import Hatchet

hatchet = Hatchet()


@hatchet.task(execution_timeout=timedelta(seconds=5))
def die(input: None, ctx: Context) -> None:
    ctx.log(f"Worker ID: {ctx.worker_id} about to die")
    ctypes.string_at(0)
    ctx.log(f"Worker ID: {ctx.worker_id} did not die")


def main(name: str) -> None:
    worker = hatchet.worker(
        name,
        workflows=[
            die,
        ],
    )

    worker.start()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--name",
        type=str,
    )

    args = parser.parse_args()

    main(str(args.name))
