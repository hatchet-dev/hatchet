from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet


class Output(BaseModel):
    message: str


hatchet = Hatchet(debug=True)


@hatchet.task(input_validator=dict)
def say_hello_unsafely(input: dict[str, str], _c: Context) -> Output:
    name = input["name"]  # untyped
    return Output(message=f"Hello, {name}!")
