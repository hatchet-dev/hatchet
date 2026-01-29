from typing import TYPE_CHECKING, Any, overload

from hatchet_sdk.context.context import Context
from hatchet_sdk.runnables.types import EmptyModel, R, TWorkflowInput
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
    ) -> Workflow[EmptyModel]: ...

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
    ) -> Workflow[EmptyModel] | Workflow[TWorkflowInput]:
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
    ) -> Standalone[EmptyModel, EmptyModel]: ...

    @overload
    def task(
        self,
        *,
        name: str,
        input_validator: None = None,
        output_validator: type[R],
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> Standalone[EmptyModel, R]: ...

    @overload
    def task(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput],
        output_validator: None = None,
        default_additional_metadata: JSONSerializableMapping | None = None,
    ) -> Standalone[TWorkflowInput, EmptyModel]: ...

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
        Standalone[EmptyModel, R]
        | Standalone[TWorkflowInput, R]
        | Standalone[EmptyModel, EmptyModel]
        | Standalone[TWorkflowInput, EmptyModel]
    ):
        def mock_func(input: Any, ctx: Context) -> Any:
            raise NotImplementedError(
                "This is a stub function and should not be called directly."
            )

        return_type = output_validator if output_validator is not None else EmptyModel
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
