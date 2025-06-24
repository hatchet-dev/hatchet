import sys
from collections.abc import Awaitable, Coroutine, Generator
from typing import Any, Literal, TypeAlias, TypeGuard, TypeVar

from pydantic import BaseModel


def is_basemodel_subclass(model: Any) -> TypeGuard[type[BaseModel]]:
    try:
        return issubclass(model, BaseModel)
    except TypeError:
        return False


class TaskIOValidator(BaseModel):
    workflow_input: type[BaseModel] | None = None
    step_output: type[BaseModel] | None = None


JSONSerializableMapping = dict[str, Any]


_T_co = TypeVar("_T_co", covariant=True)

if sys.version_info >= (3, 12):
    AwaitableLike: TypeAlias = Awaitable[_T_co]
    CoroutineLike: TypeAlias = Coroutine[Any, Any, _T_co]
else:
    AwaitableLike: TypeAlias = Generator[Any, None, _T_co] | Awaitable[_T_co]
    CoroutineLike: TypeAlias = Generator[Any, None, _T_co] | Coroutine[Any, Any, _T_co]

STOP_LOOP_TYPE = Literal["STOP_LOOP"]
STOP_LOOP: STOP_LOOP_TYPE = "STOP_LOOP"  # Sentinel object to stop the loop
