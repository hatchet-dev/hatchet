from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()


class Input(EmptyModel):
    index: int


@hatchet.task(input_validator=Input)
async def return_exceptions_task(input: Input, ctx: Context) -> dict[str, str]:
    if input.index % 2 == 0:
        raise ValueError(f"error in task with index {input.index}")

    return {"message": "this is a successful task."}


exception_parsing_workflow = hatchet.workflow(name="ExceptionParsingWorkflow")


@exception_parsing_workflow.task()
async def exception_class_no_name_task(input: EmptyModel, ctx: Context) -> None:
    class CustomNoNamedException(Exception): ...

    CustomNoNamedException.__name__ = ""
    raise CustomNoNamedException


@exception_parsing_workflow.task()
async def exception_class_task(input: EmptyModel, ctx: Context) -> None:
    raise ValueError


@exception_parsing_workflow.task()
async def exception_instance_no_args_task(input: EmptyModel, ctx: Context) -> None:
    raise ValueError()


@exception_parsing_workflow.task()
async def exception_instance_falsy_arg_task(input: EmptyModel, ctx: Context) -> None:
    raise ValueError("")


@exception_parsing_workflow.task()
async def exception_instance_truthy_arg_task(input: EmptyModel, ctx: Context) -> None:
    raise ValueError("Oh no!")
