import asyncio
from typing import Generator

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=False)

# > Streaming

anna_karenina = """
Happy families are all alike; every unhappy family is unhappy in its own way.

Everything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.
"""


def create_chunks(content: str, n: int) -> Generator[str, None, None]:
    for i in range(0, len(content), n):
        yield content[i : i + n]


chunks = list(create_chunks(anna_karenina, 10))


@hatchet.task()
async def stream_task(input: EmptyModel, ctx: Context) -> None:
    # 👀 Sleeping to avoid race conditions
    await asyncio.sleep(2)

    for chunk in chunks:
        ctx.put_stream(chunk)




def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[stream_task])
    worker.start()


if __name__ == "__main__":
    main()
