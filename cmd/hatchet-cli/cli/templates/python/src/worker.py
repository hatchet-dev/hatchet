from hatchet_client import hatchet
from workflows.first_workflow import my_task


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[my_task])
    worker.start()


if __name__ == "__main__":
    main()
