from examples.logger.client import hatchet
from examples.logger.workflow import LoggingWorkflow


def main() -> None:
    worker = hatchet.worker("logger-worker", max_runs=5)

    workflow = LoggingWorkflow()
    worker.register_workflow(workflow)

    worker.start()


if __name__ == "__main__":
    main()
