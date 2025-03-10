import asyncio

from hatchet_sdk import Hatchet

client = Hatchet()


async def main() -> None:
    client.admin.run_workflow(
        "Parent",
        {"x": "foo bar baz"},
    )


if __name__ == "__main__":
    asyncio.run(main())
