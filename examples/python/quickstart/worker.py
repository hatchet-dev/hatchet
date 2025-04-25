from .hatchet_client import hatchet
from .workflows.first_task import first_task

def main() -> None:
    worker = hatchet.worker(
        "first-worker",
        slots=10,
        workflows=[first_task],
    )
    worker.start()

if __name__ == "__main__":
    main()
