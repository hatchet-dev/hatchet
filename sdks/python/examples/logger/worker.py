from examples.logger.client import hatchet
from examples.logger.workflow import wf


def main() -> None:
    worker = hatchet.worker("logger-worker", max_runs=5, workflows=[wf])

    worker.start()


if __name__ == "__main__":
    main()
