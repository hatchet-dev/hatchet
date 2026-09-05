from datetime import timedelta
import ctypes

from hatchet_sdk import Context, EmptyModel, Hatchet

import argparse

from hatchet_sdk import Hatchet
import logging

hatchet = Hatchet()

logger = logging.getLogger(__name__)


@hatchet.task(execution_timeout=timedelta(seconds=1))
def die(input: EmptyModel, ctx: Context) -> None:
    logger.info("Worker ID: %s about to die", ctx.worker_id)
    ctypes.string_at(0)
    logger.error("Worker ID: %s did not die", ctx.worker_id)


def main(name: str) -> None:
    worker = hatchet.worker(
        name,
        slots=1,
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
