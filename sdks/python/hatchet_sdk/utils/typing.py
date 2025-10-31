import sys
from collections.abc import Awaitable, Coroutine, Generator
from dataclasses import Field as DataclassField
from dataclasses import dataclass, is_dataclass
from enum import Enum
from typing import Any, ClassVar, Literal, Protocol, TypeAlias, TypeGuard, TypeVar

from pydantic import BaseModel, SkipValidation


class DataclassInstance(Protocol):
    __dataclass_fields__: ClassVar[dict[str, DataclassField[Any]]]


def is_basemodel_subclass(model: Any) -> TypeGuard[type[BaseModel]]:
    try:
        return issubclass(model, BaseModel)
    except TypeError:
        return False


@dataclass
class PydanticModelValidator:
    validator_type: type[BaseModel]
    kind: Literal["basemodel"] = "basemodel"


@dataclass
class DataclassValidator:
    validator_type: type[DataclassInstance]
    kind: Literal["dataclass"] = "dataclass"


@dataclass
class NoValidator:
    kind: Literal["none"] = "none"


OutputValidator = PydanticModelValidator | DataclassValidator | NoValidator


def is_basemodel_validator(
    validator: OutputValidator,
) -> TypeGuard[PydanticModelValidator]:
    return validator.kind == "basemodel"


def is_dataclass_validator(validator: OutputValidator) -> TypeGuard[DataclassValidator]:
    return validator.kind == "dataclass"


def is_no_validator(validator: OutputValidator) -> TypeGuard[NoValidator]:
    return validator.kind == "none"


def classify_output_validator(return_type: Any | None) -> OutputValidator:
    if is_basemodel_subclass(return_type):
        return PydanticModelValidator(validator_type=return_type)

    if is_dataclass(return_type) and isinstance(return_type, type):
        return DataclassValidator(validator_type=return_type)

    return NoValidator()


class TaskIOValidator(BaseModel):
    workflow_input: SkipValidation[type[BaseModel] | type[DataclassInstance] | None] = (
        None
    )
    step_output: SkipValidation[type[BaseModel] | type[DataclassInstance] | None] = None


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


class LogLevel(str, Enum):
    DEBUG = "DEBUG"
    INFO = "INFO"
    WARN = "WARN"
    ERROR = "ERROR"

    @classmethod
    def from_levelname(cls, levelname: str) -> "LogLevel":
        levelname = levelname.upper()

        if levelname == "DEBUG":
            return cls.DEBUG

        if levelname == "INFO":
            return cls.INFO

        if levelname in ["WARNING", "WARN"]:
            return cls.WARN

        if levelname == "ERROR":
            return cls.ERROR

        # fall back to INFO
        return cls.INFO
