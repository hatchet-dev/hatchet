import json
import traceback
from typing import cast


class InvalidDependencyError(Exception):
    pass


class NonRetryableException(Exception):  # noqa: N818
    pass


class DedupeViolationError(Exception):
    """Raised by the Hatchet library to indicate that a workflow has already been run with this deduplication value."""


TASK_RUN_ERROR_METADATA_KEY = "__hatchet_error_metadata__"


class TaskRunError(Exception):
    def __init__(
        self,
        exc: str,
        exc_type: str,
        trace: str,
        task_run_external_id: str | None,
    ) -> None:
        self.exc = exc
        self.exc_type = exc_type
        self.trace = trace
        self.task_run_external_id = task_run_external_id

    def __str__(self) -> str:
        return self.serialize(include_metadata=False)

    def __repr__(self) -> str:
        return str(self)

    def serialize(self, include_metadata: bool) -> str:
        if not self.exc_type or not self.exc:
            return ""

        metadata = json.dumps(
            {
                TASK_RUN_ERROR_METADATA_KEY: {
                    "task_run_external_id": self.task_run_external_id,
                }
            },
            indent=None,
        )

        result = (
            self.exc_type.replace(": ", ":::")
            + ": "
            + self.exc.replace("\n", "\\\n")
            + "\n"
            + self.trace
        )

        if include_metadata:
            return result + "\n\n" + metadata

        return result

    @classmethod
    def _extract_metadata(cls, serialized: str) -> tuple[str, dict[str, str | None]]:
        metadata = serialized.split("\n")[-1]

        try:
            parsed = json.loads(metadata)

            if (
                TASK_RUN_ERROR_METADATA_KEY in parsed
                and "task_run_external_id" in parsed[TASK_RUN_ERROR_METADATA_KEY]
            ):
                serialized = serialized.replace(metadata, "").strip()
                return serialized, cast(
                    dict[str, str | None], parsed[TASK_RUN_ERROR_METADATA_KEY]
                )

            return serialized, {}
        except json.JSONDecodeError:
            return serialized, {}

    @classmethod
    def _unpack_serialized_error(cls, serialized: str) -> tuple[str | None, str, str]:
        serialized, metadata = cls._extract_metadata(serialized)

        external_id = metadata.get("task_run_external_id", None)
        header, trace = serialized.split("\n", 1)

        return external_id, header, trace

    @classmethod
    def deserialize(cls, serialized: str) -> "TaskRunError":
        if not serialized:
            return cls(
                exc="",
                exc_type="",
                trace="",
                task_run_external_id=None,
            )

        task_run_external_id = None

        try:
            task_run_external_id, header, trace = cls._unpack_serialized_error(
                serialized
            )

            exc_type, exc = header.split(": ", 1)
        except ValueError:
            ## If we get here, we saw an error that was not serialized how we expected,
            ## but was also not empty. So we return it as-is and use `HatchetError` as the type.
            return cls(
                exc=serialized,
                exc_type="HatchetError",
                trace="",
                task_run_external_id=task_run_external_id,
            )

        exc_type = exc_type.replace(":::", ": ")
        exc = exc.replace("\\\n", "\n")

        return cls(
            exc=exc,
            exc_type=exc_type,
            trace=trace,
            task_run_external_id=task_run_external_id,
        )

    @classmethod
    def from_exception(
        cls, exc: Exception, task_run_external_id: str | None
    ) -> "TaskRunError":
        return cls(
            exc=str(exc),
            exc_type=type(exc).__name__,
            trace="".join(
                traceback.format_exception(type(exc), exc, exc.__traceback__)
            ),
            task_run_external_id=task_run_external_id,
        )


class FailedTaskRunExceptionGroup(ValueError):  # noqa: N818
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


class LifespanSetupError(Exception):
    pass
