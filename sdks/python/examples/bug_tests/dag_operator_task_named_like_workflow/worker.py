from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()

WORKFLOW_NAME = "task-named-like-workflow"
TASK_NAMED_LIKE_WORKFLOW = hatchet.namespace + WORKFLOW_NAME

task_named_like_workflow_dag = hatchet.workflow(name=WORKFLOW_NAME)


@task_named_like_workflow_dag.task(name=TASK_NAMED_LIKE_WORKFLOW)
async def first_step(input: EmptyModel, ctx: Context) -> None:
    pass


@task_named_like_workflow_dag.task(parents=[first_step])
async def second_step(input: EmptyModel, ctx: Context) -> None:
    pass


def main() -> None:
    worker = hatchet.worker(
        "task-named-like-workflow-worker",
        workflows=[task_named_like_workflow_dag],
    )
    worker.start()


if __name__ == "__main__":
    main()
