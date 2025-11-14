from dataclasses import dataclass
from typing import Literal

from hatchet_sdk import Context, EmptyModel, Hatchet


# > Dataclasses
@dataclass
class Input:
    name: str


@dataclass
class Output:
    message: str


# !!


hatchet = Hatchet(debug=True)


# > Task using dataclasses
@hatchet.task(input_validator=Input)
def say_hello(input: Input, ctx: Context) -> Output:
    return Output(message=f"Hello, {input.name}!")


# !!


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[say_hello])
    worker.start()


if __name__ == "__main__":
    main()
