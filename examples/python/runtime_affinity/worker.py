import argparse

from hatchet_sdk import Context, EmptyModel, Hatchet, WorkerLabel
from pydantic import BaseModel

hatchet = Hatchet()


class AffinityResult(BaseModel):
    worker_id: str


runtime_affinity_workflow = hatchet.workflow(name="runtime_affinity_workflow")


@runtime_affinity_workflow.task()
async def affinity_task_1(i: EmptyModel, c: Context) -> AffinityResult:
    return AffinityResult(worker_id=c.worker_id)


@runtime_affinity_workflow.task(parents=[affinity_task_1])
async def affinity_task_2(i: EmptyModel, c: Context) -> AffinityResult:
    return AffinityResult(worker_id=c.worker_id)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--label", type=str, required=True)
    args = parser.parse_args()

    worker = hatchet.worker(
        "runtime-affinity-worker",
        labels={"affinity": args.label},
        workflows=[runtime_affinity_workflow],
    )

    worker.start()


if __name__ == "__main__":
    main()
