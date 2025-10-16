from collections.abc import Generator
from typing import TypeVar

T = TypeVar("T")


def create_chunks(xs: list[T], n: int) -> Generator[list[T], None, None]:
    for i in range(0, len(xs), n):
        yield xs[i : i + n]
