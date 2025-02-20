import asyncio

from hatchet_sdk import new_client


async def main() -> None:

    hatchet = new_client()

    hatchet.admin.run_workflow(
        "Parent",
        {"x": "foo bar baz"},
    )


if __name__ == "__main__":
    asyncio.run(main())
