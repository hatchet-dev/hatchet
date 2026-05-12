from typing import TYPE_CHECKING, Any, overload

from hatchet_sdk.context.context import Context
from hatchet_sdk.runnables.types import R, TWorkflowInput
from hatchet_sdk.runnables.workflow import Standalone, Workflow
from hatchet_sdk.utils.typing import JSONSerializableMapping

if TYPE_CHECKING:
    from hatchet_sdk import Hatchet


class StubsClient:
    def __init__(self, client: "Hatchet"):
        self.client = client

    @overload
    def workflow(
        self,
        *,
        name: str,
        input_validator: None = None,
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> Workflow[None]: ...

    @overload
    def workflow(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput],
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> Workflow[TWorkflowInput]: ...

    def workflow(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput] | None = None,
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> Workflow[None] | Workflow[TWorkflowInput]:
        return self.client.workflow(
            name=name,
            input_validator=input_validator,
            default_additional_metadata=default_additional_metadata,
        )

    @overload
    def task(
        self,
        *,
        name: str,
        input_validator: None = None,
        output_validator: None = None,
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> Standalone[None, None]: ...

    @overload
    def task(
        self,
        *,
        name: str,
        input_validator: None = None,
        output_validator: type[R],
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> Standalone[None, R]: ...

    @overload
    def task(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput],
        output_validator: None = None,
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> Standalone[TWorkflowInput, None]: ...

    @overload
    def task(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput],
        output_validator: type[R],
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> Standalone[TWorkflowInput, R]: ...

    def task(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput] | None = None,
        output_validator: type[R] | None = None,
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> (
        Standalone[None, R]
        | Standalone[TWorkflowInput, R]
        | Standalone[None, None]
        | Standalone[TWorkflowInput, None]
    ):
        def mock_func(input: Any, ctx: Context) -> Any:
            raise NotImplementedError(
                "This is a stub function and should not be called directly."
            )

        return_type = output_validator if output_validator is not None else None
        mock_func.__annotations__ = {
            "input": Any,
            "ctx": Context,
            "return": return_type,
        }

        return self.client.task(
            name=name,
            input_validator=input_validator,
            default_additional_metadata=default_additional_metadata,
        )(mock_func)
