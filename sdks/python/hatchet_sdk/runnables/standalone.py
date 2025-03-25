import asyncio
from typing import Any, Generic, cast, get_type_hints

from hatchet_sdk.clients.admin import TriggerWorkflowOptions, WorkflowRunTriggerConfig
from hatchet_sdk.runnables.types import R, TWorkflowInput
from hatchet_sdk.runnables.workflow import Workflow
from hatchet_sdk.utils.aio_utils import get_active_event_loop
from hatchet_sdk.utils.typing import is_basemodel_subclass
from hatchet_sdk.workflow_run import WorkflowRunRef


class TaskRunRef(Generic[TWorkflowInput, R]):
    def __init__(
        self,
        standalone: "Standalone[TWorkflowInput, R]",
        workflow_run_ref: WorkflowRunRef,
    ):
        self._s = standalone
        self._wrr = workflow_run_ref

    async def aio_result(self) -> R:
        result = await self._wrr.workflow_listener.result(self._wrr.workflow_run_id)
        return self._s._extract_result(result)

    def result(self) -> R:
        coro = self._wrr.workflow_listener.result(self._wrr.workflow_run_id)

        loop = get_active_event_loop()

        if loop is None:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            try:
                result = loop.run_until_complete(coro)
            finally:
                asyncio.set_event_loop(None)
        else:
            result = loop.run_until_complete(coro)

        return self._s._extract_result(result)


class Standalone(Generic[TWorkflowInput, R]):
    def __init__(self, workflow: Workflow[TWorkflowInput]) -> None:
        self._workflow = workflow
        self._task = workflow.tasks[0]

        return_type = get_type_hints(self._task.fn).get("return")

        self._output_validator = (
            return_type if is_basemodel_subclass(return_type) else None
        )

    def _extract_result(self, result: dict[str, Any]) -> R:
        output = result.get(self._task.name)

        if not output:
            raise ValueError(f"Task {self._task.name} did not return any output")

        if not self._output_validator:
            return cast(R, output)

        return cast(R, self._output_validator.model_validate(output))

    def run(
        self,
        input: TWorkflowInput | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> R:
        return self._extract_result(self._workflow.run(input, options))

    async def aio_run(
        self,
        input: TWorkflowInput | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> R:
        result = await self._workflow.aio_run(input, options)
        return self._extract_result(result)

    def run_no_wait(
        self,
        input: TWorkflowInput | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> TaskRunRef[TWorkflowInput, R]:
        ref = self._workflow.run_no_wait(input, options)

        return TaskRunRef[TWorkflowInput, R](self, ref)

    async def aio_run_no_wait(
        self,
        input: TWorkflowInput | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> TaskRunRef[TWorkflowInput, R]:
        ref = await self._workflow.aio_run_no_wait(input, options)

        return TaskRunRef[TWorkflowInput, R](self, ref)

    def run_many(self, workflows: list[WorkflowRunTriggerConfig]) -> list[R]:
        return [
            self._extract_result(result)
            for result in self._workflow.run_many(workflows)
        ]

    async def aio_run_many(self, workflows: list[WorkflowRunTriggerConfig]) -> list[R]:
        return [
            self._extract_result(result)
            for result in await self._workflow.aio_run_many(workflows)
        ]

    def run_many_no_wait(
        self, workflows: list[WorkflowRunTriggerConfig]
    ) -> list[TaskRunRef[TWorkflowInput, R]]:
        refs = self._workflow.run_many_no_wait(workflows)

        return [TaskRunRef[TWorkflowInput, R](self, ref) for ref in refs]

    async def aio_run_many_no_wait(
        self, workflows: list[WorkflowRunTriggerConfig]
    ) -> list[TaskRunRef[TWorkflowInput, R]]:
        refs = await self._workflow.aio_run_many_no_wait(workflows)

        return [TaskRunRef[TWorkflowInput, R](self, ref) for ref in refs]
