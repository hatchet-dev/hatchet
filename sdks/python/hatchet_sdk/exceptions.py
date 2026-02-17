import json
import traceback
from enum import Enum
from typing import cast


class NonDeterminismError(Exception):
    def __init__(
        self, task_external_id: str, invocation_count: int, message: str, node_id: int
    ) -> None:
        self.task_external_id = task_external_id
        self.invocation_count = invocation_count
        self.message = message
        self.node_id = node_id

        super().__init__(
            f"Non-determinism detected in task {task_external_id} on invocation {invocation_count} at node {node_id}.\nCheck out our documentation for more details on expectations of durable tasks: https://docs.hatchet.run/home/durable-best-practices"
        )

    def serialize(self, include_metadata: bool = False) -> str:
        return str(self)

    @property
    def exc(self) -> str:
        return self.message


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
        exc_type = self.exc_type.replace(": ", ":::")
        exc = self.exc.replace("\n", "\\\n")
        header = f"{exc_type}: {exc}" if exc_type and exc else f"{exc_type}{exc}"
        result = (
            f"{header}\n{self.trace}"
            if header and self.trace
            else f"{header}{self.trace}"
        )
        if result == "":
            return result

        if include_metadata:
            metadata = json.dumps(
                {
                    TASK_RUN_ERROR_METADATA_KEY: {
                        "task_run_external_id": self.task_run_external_id,
                    }
                },
                indent=None,
            )
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
    def __init__(
        self, message: str, exceptions: list[TaskRunError | NonDeterminismError]
    ):
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


class CancellationReason(Enum):
    """Reason for cancellation of an operation."""

    USER_REQUESTED = "user_requested"
    """The user explicitly requested cancellation."""

    TIMEOUT = "timeout"
    """The operation timed out."""

    PARENT_CANCELLED = "parent_cancelled"
    """The parent workflow or task was cancelled."""

    WORKFLOW_CANCELLED = "workflow_cancelled"
    """The workflow run was cancelled."""

    TOKEN_CANCELLED = "token_cancelled"
    """The cancellation token was cancelled."""


class CancelledError(BaseException):
    """
    Raised when an operation is cancelled via CancellationToken.

    This exception inherits from BaseException (not Exception) so that it
    won't be caught by bare `except Exception:` handlers. This mirrors the
    behavior of asyncio.CancelledError in Python 3.8+.

    To catch this exception, use:
        - `except CancelledError:` (recommended)
        - `except BaseException:` (catches all exceptions)

    This exception is used for sync code paths. For async code paths,
    asyncio.CancelledError is used instead.

    :param message: Optional message describing the cancellation.
    :param reason: Optional enum indicating the reason for cancellation.
    """

    def __init__(
        self,
        message: str = "Operation cancelled",
        reason: CancellationReason | None = None,
    ) -> None:
        self.reason = reason
        super().__init__(message)

    @property
    def message(self) -> str:
        return str(self.args[0]) if self.args else "Operation cancelled"
