from datetime import datetime
from typing import Any, Generic, cast, get_type_hints

from hatchet_sdk.clients.admin import (
    ScheduleTriggerWorkflowOptions,
    TriggerWorkflowOptions,
    WorkflowRunTriggerConfig,
)
from hatchet_sdk.clients.rest.models.cron_workflows import CronWorkflows
from hatchet_sdk.contracts.workflows_pb2 import WorkflowVersion
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.types import EmptyModel, R, TWorkflowInput
from hatchet_sdk.runnables.workflow import BaseWorkflow, Workflow
from hatchet_sdk.utils.aio import run_async_from_sync
from hatchet_sdk.utils.typing import JSONSerializableMapping, is_basemodel_subclass
from hatchet_sdk.workflow_run import WorkflowRunRef


class TaskRunRef(Generic[TWorkflowInput, R]):
    def __init__(
        self,
        standalone: "Standalone[TWorkflowInput, R]",
        workflow_run_ref: WorkflowRunRef,
    ):
        self._s = standalone
        self._wrr = workflow_run_ref

        self.workflow_run_id = workflow_run_ref.workflow_run_id

    def __str__(self) -> str:
        return self.workflow_run_id

    async def aio_result(self) -> R:
        result = await self._wrr.workflow_listener.aio_result(self._wrr.workflow_run_id)
        return self._s._extract_result(result)

    def result(self) -> R:
        return run_async_from_sync(self.aio_result)


class Standalone(BaseWorkflow[TWorkflowInput], Generic[TWorkflowInput, R]):
    def __init__(
        self, workflow: Workflow[TWorkflowInput], task: Task[TWorkflowInput, R]
    ) -> None:
        super().__init__(config=workflow.config, client=workflow.client)

        ## NOTE: This is a hack to assign the task back to the base workflow,
        ## since the decorator to mutate the tasks is not being called.
        self._default_tasks = [task]

        self._workflow = workflow
        self._task = task

        return_type = get_type_hints(self._task.fn).get("return")

        self._output_validator = (
            return_type if is_basemodel_subclass(return_type) else None
        )

        self.config = self._workflow.config

    def _extract_result(self, result: dict[str, Any]) -> R:
        output = result.get(self._task.name)

        if not self._output_validator:
            return cast(R, output)

        return cast(R, self._output_validator.model_validate(output))

    def run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> R:
        return self._extract_result(self._workflow.run(input, options))

    async def aio_run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> R:
        result = await self._workflow.aio_run(input, options)
        return self._extract_result(result)

    def run_no_wait(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> TaskRunRef[TWorkflowInput, R]:
        ref = self._workflow.run_no_wait(input, options)

        return TaskRunRef[TWorkflowInput, R](self, ref)

    async def aio_run_no_wait(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
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

    def schedule(
        self,
        run_at: datetime,
        input: TWorkflowInput | None = None,
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
        return self._workflow.schedule(
            run_at=run_at,
            input=input,
            options=options,
        )

    async def aio_schedule(
        self,
        run_at: datetime,
        input: TWorkflowInput,
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
        return await self._workflow.aio_schedule(
            run_at=run_at,
            input=input,
            options=options,
        )

    def create_cron(
        self,
        cron_name: str,
        expression: str,
        input: TWorkflowInput,
        additional_metadata: JSONSerializableMapping,
    ) -> CronWorkflows:
        return self._workflow.create_cron(
            cron_name=cron_name,
            expression=expression,
            input=input,
            additional_metadata=additional_metadata,
        )

    async def aio_create_cron(
        self,
        cron_name: str,
        expression: str,
        input: TWorkflowInput,
        additional_metadata: JSONSerializableMapping,
    ) -> CronWorkflows:
        return await self._workflow.aio_create_cron(
            cron_name=cron_name,
            expression=expression,
            input=input,
            additional_metadata=additional_metadata,
        )

    def to_task(self) -> Task[TWorkflowInput, R]:
        return self._task
