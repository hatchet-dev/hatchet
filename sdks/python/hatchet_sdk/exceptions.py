import json
import traceback
from typing import cast


class NonRetryableException(Exception):  # noqa: N818
    pass


class DedupeViolationError(Exception):
    """Raised by the Hatchet library to indicate that a workflow has already been run with this deduplication value."""


class FailedTaskRunError(Exception):
    def __init__(
        self,
        exc: str,
        exc_type: str,
        trace: str,
    ) -> None:
        self.exc = exc
        self.exc_type = exc_type
        self.trace = trace

    def __str__(self) -> str:
        return f"{self.exc}\n{self.trace}"

    def __repr__(self) -> str:
        return str(self)

    def serialize(self) -> str:
        return json.dumps(
            {
                "exc": self.exc,
                "exc_type": self.exc_type,
                "trace": self.trace,
            }
        )

    @classmethod
    def deserialize(cls, serialized: str) -> "FailedTaskRunError":
        parsed = cast(dict[str, str], json.loads(serialized))

        return cls(
            exc=parsed.get("exc", ""),
            exc_type=parsed.get("exc_type", ""),
            trace=parsed.get("trace", ""),
        )

    @classmethod
    def from_exception(cls, exc: Exception) -> "FailedTaskRunError":
        return cls(
            exc=str(exc),
            exc_type=type(exc).__name__,
            trace="".join(
                traceback.format_exception(type(exc), exc, exc.__traceback__)
            ),
        )


class FailedTaskRunExceptionGroup(Exception):  # noqa: N818
    def __init__(self, message: str, exceptions: list[FailedTaskRunError]):
        self.message = message
        self.exceptions = exceptions

        super().__init__(message)

    def __str__(self) -> str:
        result = [self.message]

        for i, exc in enumerate(self.exceptions, 1):
            result.append(f"\n--- Exception {i} ---")
            result.append(str(exc))

        return "\n".join(result)


class LoopAlreadyRunningError(Exception):
    pass
