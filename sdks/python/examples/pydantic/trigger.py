import asyncio

from examples.pydantic.worker import ParentInput, parent_workflow
from hatchet_sdk import Hatchet

client = Hatchet()


async def main() -> None:
    parent_workflow.run(ParentInput(x="foo bar baz"))


if __name__ == "__main__":
    asyncio.run(main())
