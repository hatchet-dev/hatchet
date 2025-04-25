from hatchet_sdk import Hatchet
from tests.correct_failure_on_timeout_with_multi_concurrency.workflow import (
    multiple_concurrent_cancellations_test_workflow,
)

hatchet = Hatchet(debug=True)


def main() -> None:
    worker = hatchet.worker(
        "e2e-test-worker-2",
        slots=100,
        workflows=[multiple_concurrent_cancellations_test_workflow],
    )

    worker.start()


if __name__ == "__main__":
    main()
