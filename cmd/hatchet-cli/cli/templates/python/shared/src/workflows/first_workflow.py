from hatchet_sdk import Context

from hatchet_client import hatchet


# Declare the task to run
@hatchet.task(name="first-workflow")
def my_task(input: None, ctx: Context) -> dict[str, int]:
    print("executed task")

    return {"meaning_of_life": 42}
