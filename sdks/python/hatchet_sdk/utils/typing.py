from typing import Any, Mapping, Type, TypeGuard

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
