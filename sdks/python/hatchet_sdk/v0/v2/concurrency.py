from typing import Any, Callable

from hatchet_sdk.contracts.workflows_pb2 import (  # type: ignore[attr-defined]
    ConcurrencyLimitStrategy,
)
from hatchet_sdk.v0.context.context import Context


class ConcurrencyFunction:
    def __init__(
        self,
        func: Callable[[Context], str],
        name: str = "concurrency",
        max_runs: int = 1,
        limit_strategy: ConcurrencyLimitStrategy = ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ):
        self.func = func
        self.name = name
        self.max_runs = max_runs
        self.limit_strategy = limit_strategy
        self.namespace = "default"

    def set_namespace(self, namespace: str) -> None:
        self.namespace = namespace

    def get_action_name(self) -> str:
        return self.namespace + ":" + self.name

    def __call__(self, *args: Any, **kwargs: Any) -> str:
        return self.func(*args, **kwargs)

    def __str__(self) -> str:
        return f"{self.name}({self.max_runs})"

    def __repr__(self) -> str:
        return f"{self.name}({self.max_runs})"


def concurrency(
    name: str = "",
    max_runs: int = 1,
    limit_strategy: ConcurrencyLimitStrategy = ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
) -> Callable[[Callable[[Context], str]], ConcurrencyFunction]:
    def inner(func: Callable[[Context], str]) -> ConcurrencyFunction:
        return ConcurrencyFunction(func, name, max_runs, limit_strategy)

    return inner
