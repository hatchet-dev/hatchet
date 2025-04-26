import argparse
from typing import cast

from hatchet_sdk import Hatchet
from tests.correct_failure_on_timeout_with_multi_concurrency.workflow import (
    multiple_concurrent_cancellations_test_workflow,
)

hatchet = Hatchet(debug=True)


def main(slots: int) -> None:
    worker = hatchet.worker(
        "e2e-test-worker-2",
        slots=slots,
        workflows=[multiple_concurrent_cancellations_test_workflow],
    )

    worker.start()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--slots",
        type=int,
        default=100,
        required=False,
    )

    args = parser.parse_args()

    slots = cast(int | None, args.slots) or 100

    main(slots)
