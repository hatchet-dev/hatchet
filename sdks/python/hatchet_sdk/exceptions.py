import traceback


class NonRetryableException(Exception):  # noqa: N818
    pass


class DedupeViolationError(Exception):
    """Raised by the Hatchet library to indicate that a workflow has already been run with this deduplication value."""


class TaskRunError(Exception):
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
        return self.serialize()

    def __repr__(self) -> str:
        return str(self)

    def serialize(self) -> str:
        if not self.exc_type or not self.exc:
            return ""

        return (
            self.exc_type.replace(": ", ":::")
            + ": "
            + self.exc.replace("\n", "\\\n")
            + "\n"
            + self.trace
        )

    @classmethod
    def deserialize(cls, serialized: str) -> "TaskRunError":
        if not serialized:
            return cls(
                exc="",
                exc_type="",
                trace="",
            )

        try:
            header, trace = serialized.split("\n", 1)
            exc_type, exc = header.split(": ", 1)
        except ValueError:
            ## If we get here, we saw an error that was not serialized how we expected,
            ## but was also not empty. So we return it as-is and use `HatchetError` as the type.
            return cls(
                exc=serialized,
                exc_type="HatchetError",
                trace="",
            )

        exc_type = exc_type.replace(":::", ": ")
        exc = exc.replace("\\\n", "\n")

        return cls(
            exc=exc,
            exc_type=exc_type,
            trace=trace,
        )

    @classmethod
    def from_exception(cls, exc: Exception) -> "TaskRunError":
        return cls(
            exc=str(exc),
            exc_type=type(exc).__name__,
            trace="".join(
                traceback.format_exception(type(exc), exc, exc.__traceback__)
            ),
        )


class FailedTaskRunExceptionGroup(Exception):  # noqa: N818
    def __init__(self, message: str, exceptions: list[TaskRunError]):
        self.message = message
        self.exceptions = exceptions

        super().__init__(message)

    def __str__(self) -> str:
        result = [self.message.strip()]

        for i, exc in enumerate(self.exceptions, 1):
            result.append(f"\n--- Exception {i} ---")
            result.append(str(exc))

        return "\n".join(result)


class LoopAlreadyRunningError(Exception):
    pass


class IllegalTaskOutputError(Exception):
    pass
