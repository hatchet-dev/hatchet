from hatchet_sdk import Context, EmptyModel, Hatchet, TriggerWorkflowOptions

hatchet = Hatchet()


@hatchet.task()
async def spawned_child(input: EmptyModel, ctx: Context) -> None:
    pass


parent_dag = hatchet.workflow(name="colliding-dag")


@parent_dag.task()
async def step_a(input: EmptyModel, ctx: Context) -> None:
    ref = await spawned_child.aio_run_no_wait()


@parent_dag.task(parents=[step_a])
async def step_b(input: EmptyModel, ctx: Context) -> None:
    pass
