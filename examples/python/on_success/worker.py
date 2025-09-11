from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

on_success_workflow = hatchet.workflow(name="OnSuccessWorkflow")


@on_success_workflow.task()
def first_task(input: EmptyModel, ctx: Context) -> None:
    print("First task completed successfully")


@on_success_workflow.task(parents=[first_task])
def second_task(input: EmptyModel, ctx: Context) -> None:
    print("Second task completed successfully")


@on_success_workflow.task(parents=[first_task, second_task])
def third_task(input: EmptyModel, ctx: Context) -> None:
    print("Third task completed successfully")


@on_success_workflow.task()
def fourth_task(input: EmptyModel, ctx: Context) -> None:
    print("Fourth task completed successfully")


@on_success_workflow.on_success_task()
def on_success_task(input: EmptyModel, ctx: Context) -> None:
    print("On success task completed successfully")


def main() -> None:
    worker = hatchet.worker("on-success-worker", workflows=[on_success_workflow])
    worker.start()


if __name__ == "__main__":
    main()
