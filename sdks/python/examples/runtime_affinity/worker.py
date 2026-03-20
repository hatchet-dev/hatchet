import argparse

from hatchet_sdk import Context, EmptyModel, Hatchet, WorkerLabel
from pydantic import BaseModel

hatchet = Hatchet()


class AffinityResult(BaseModel):
    worker_id: str


@hatchet.task()
async def affinity_example_task(i: EmptyModel, c: Context) -> AffinityResult:
    return AffinityResult(worker_id=c.worker_id)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--label", type=str, required=True)
    args = parser.parse_args()

    worker = hatchet.worker(
        "runtime-affinity-worker",
        labels=[WorkerLabel(key="affinity", value=args.label)],
        workflows=[affinity_example_task],
    )

    worker.start()


if __name__ == "__main__":
    main()
