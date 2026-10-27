from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()


class ParentOutput(BaseModel):
    retry_count: int


class ChildOutput(BaseModel):
    parent_retry_count: int
    retry_count: int


dag_replay_child_reads_parent_output = hatchet.workflow(
    name="dag-replay-child-reads-parent-output"
)


@dag_replay_child_reads_parent_output.task()
async def parent_failing_on_first_attempt(
    input: EmptyModel, ctx: Context
) -> ParentOutput:
    if ctx.retry_count == 0:
        raise ValueError("fails on the first attempt and succeeds when replayed")

    return ParentOutput(retry_count=ctx.retry_count)


@dag_replay_child_reads_parent_output.task(parents=[parent_failing_on_first_attempt])
async def child_reading_parent_output(input: EmptyModel, ctx: Context) -> ChildOutput:
    parent_output = ParentOutput.model_validate(
        ctx.task_output(parent_failing_on_first_attempt)
    )

    return ChildOutput(
        parent_retry_count=parent_output.retry_count, retry_count=ctx.retry_count
    )


def main() -> None:
    worker = hatchet.worker(
        "dag-replay-child-reads-parent-output-worker",
        workflows=[dag_replay_child_reads_parent_output],
    )
    worker.start()


if __name__ == "__main__":
    main()
