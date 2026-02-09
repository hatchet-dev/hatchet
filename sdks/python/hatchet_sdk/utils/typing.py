import sys
from collections.abc import Awaitable, Coroutine, Generator
from dataclasses import Field as DataclassField
from enum import Enum
from typing import Any, ClassVar, Literal, Protocol, TypeAlias, TypeVar


class DataclassInstance(Protocol):
    __dataclass_fields__: ClassVar[dict[str, DataclassField[Any]]]


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
