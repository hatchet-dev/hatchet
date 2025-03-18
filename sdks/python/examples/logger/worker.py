from examples.logger.client import hatchet
from examples.logger.workflow import logging_workflow

def main() -> None:
    worker = hatchet.worker("logger-worker", slots=5, workflows=[logging_workflow])

    worker.start()


if __name__ == "__main__":
    main()
