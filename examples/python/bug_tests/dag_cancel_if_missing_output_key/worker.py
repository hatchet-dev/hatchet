from hatchet_sdk import Context, EmptyModel, Hatchet, ParentCondition

hatchet = Hatchet()

cancel_if_missing_output_key_dag = hatchet.workflow(
    name="dag-cancel-if-missing-output-key"
)


@cancel_if_missing_output_key_dag.task()
async def parent_without_flag_key(input: EmptyModel, ctx: Context) -> dict[str, int]:
    return {"value": 1}


@cancel_if_missing_output_key_dag.task(
    parents=[parent_without_flag_key],
    cancel_if=[
        ParentCondition(
            parent=parent_without_flag_key,
            expression="output.flag == true",
        )
    ],
)
async def child_with_cancel_condition(
    input: EmptyModel, ctx: Context
) -> dict[str, bool]:
    return {"ran": True}
