from typing import TYPE_CHECKING, overload

from hatchet_sdk.context.context import Context
from hatchet_sdk.runnables.types import EmptyModel, R, TWorkflowInput
from hatchet_sdk.runnables.workflow import Standalone, Workflow

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
    ) -> Workflow[EmptyModel]: ...

    @overload
    def workflow(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput],
    ) -> Workflow[TWorkflowInput]: ...

    def workflow(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput] | None = None,
    ) -> Workflow[EmptyModel] | Workflow[TWorkflowInput]:
        return self.client.workflow(name=name, input_validator=input_validator)

    @overload
    def task(
        self,
        *,
        name: str,
        input_validator: None = None,
    ) -> Standalone[EmptyModel, R]: ...

    @overload
    def task(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput],
    ) -> Standalone[TWorkflowInput, R]: ...

    def task(
        self,
        *,
        name: str,
        input_validator: type[TWorkflowInput] | None = None,
    ) -> Standalone[EmptyModel, R] | Standalone[TWorkflowInput, R]:
        def mock_func(input: TWorkflowInput, ctx: Context) -> R:
            raise NotImplementedError(
                "This is a stub function and should not be called directly."
            )

        return self.client.task(name=name, input_validator=input_validator)(mock_func)
