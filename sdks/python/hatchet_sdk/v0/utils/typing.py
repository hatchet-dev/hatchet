from typing import Any, Type, TypeGuard, TypeVar

from pydantic import BaseModel

T = TypeVar("T", bound=BaseModel)


def is_basemodel_subclass(model: Any) -> bool:
    try:
        return issubclass(model, BaseModel)
    except TypeError:
        return False
