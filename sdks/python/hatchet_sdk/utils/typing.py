import sys
from typing import (
    Any,
    Awaitable,
    Coroutine,
    Generator,
    Mapping,
    Type,
    TypeAlias,
    TypeGuard,
    TypeVar,
)

from pydantic import BaseModel


def is_basemodel_subclass(model: Any) -> TypeGuard[Type[BaseModel]]:
    try:
        return issubclass(model, BaseModel)
    except TypeError:
        return False


class TaskIOValidator(BaseModel):
    workflow_input: Type[BaseModel] | None = None
    step_output: Type[BaseModel] | None = None


JSONSerializableMapping = Mapping[str, Any]


_T_co = TypeVar("_T_co", covariant=True)

if sys.version_info >= (3, 12):
    AwaitableLike: TypeAlias = Awaitable[_T_co]  # noqa: Y047
    CoroutineLike: TypeAlias = Coroutine[Any, Any, _T_co]  # noqa: Y047
else:
    AwaitableLike: TypeAlias = Generator[Any, None, _T_co] | Awaitable[_T_co]
    CoroutineLike: TypeAlias = Generator[Any, None, _T_co] | Coroutine[Any, Any, _T_co]
