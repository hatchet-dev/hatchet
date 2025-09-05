# > Worker
import asyncio

from aiohttp import ClientSession

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()


async def fetch(session: ClientSession, url: str) -> bool:
    async with session.get(url) as response:
        return response.status == 200


@hatchet.task(name="Fetch")
async def hello_from_hatchet(input: EmptyModel, ctx: Context) -> dict[str, int]:
    num_requests = 10

    async with ClientSession() as session:
        tasks = [
            fetch(session, "https://docs.hatchet.run/home") for _ in range(num_requests)
        ]

        results = await asyncio.gather(*tasks)

        return {"count": results.count(True)}


