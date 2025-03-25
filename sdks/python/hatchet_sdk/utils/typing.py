from typing import Any, Mapping, Type, TypeGuard, TypeVar

from pydantic import BaseModel

T = TypeVar("T", bound=BaseModel)


def is_basemodel_subclass(model: Any) -> TypeGuard[Type[BaseModel]]:
    try:
        return issubclass(model, BaseModel)
    except TypeError:
        return False


class WorkflowValidator(BaseModel):
    workflow_input: Type[BaseModel] | None = None
    step_output: Type[BaseModel] | None = None


JSONSerializableMapping = Mapping[str, Any]
