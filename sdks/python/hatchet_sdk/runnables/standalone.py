from typing import Any, Generic, cast, get_type_hints

from hatchet_sdk.clients.admin import TriggerWorkflowOptions, WorkflowRunTriggerConfig
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.types import EmptyModel, R, TWorkflowInput
from hatchet_sdk.runnables.workflow import BaseWorkflow, Workflow
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

        self.workflow_run_id = workflow_run_ref.workflow_run_id

    def __str__(self) -> str:
        return self.workflow_run_id

    async def aio_result(self) -> R:
        result = await self._wrr.workflow_run_listener.aio_result(
            self._wrr.workflow_run_id
        )
        return self._s._extract_result(result)

    def result(self) -> R:
        result = self._wrr.result()

        return self._s._extract_result(result)


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
        """
        Synchronously trigger a workflow run without waiting for it to complete.
        This method is useful for starting a workflow run and immediately returning a reference to the run without blocking while the workflow runs.

        :param input: The input data for the workflow.
        :param options: Additional options for workflow execution.

        :returns: A `WorkflowRunRef` object representing the reference to the workflow run.
        """
        return self._extract_result(self._workflow.run(input, options))

    async def aio_run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> R:
        """
        Run the workflow asynchronously and wait for it to complete.

        This method triggers a workflow run, blocks until completion, and returns the final result.

        :param input: The input data for the workflow, must match the workflow's input type.
        :param options: Additional options for workflow execution like metadata and parent workflow ID.

        :returns: The result of the workflow execution as a dictionary.
        """
        result = await self._workflow.aio_run(input, options)
        return self._extract_result(result)

    def run_no_wait(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> TaskRunRef[TWorkflowInput, R]:
        """
        Run the workflow synchronously and wait for it to complete.

        This method triggers a workflow run, blocks until completion, and returns the final result.

        :param input: The input data for the workflow, must match the workflow's input type.
        :param options: Additional options for workflow execution like metadata and parent workflow ID.

        :returns: The result of the workflow execution as a dictionary.
        """
        ref = self._workflow.run_no_wait(input, options)

        return TaskRunRef[TWorkflowInput, R](self, ref)

    async def aio_run_no_wait(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> TaskRunRef[TWorkflowInput, R]:
        """
        Asynchronously trigger a workflow run without waiting for it to complete.
        This method is useful for starting a workflow run and immediately returning a reference to the run without blocking while the workflow runs.

        :param input: The input data for the workflow.
        :param options: Additional options for workflow execution.

        :returns: A `WorkflowRunRef` object representing the reference to the workflow run.
        """
        ref = await self._workflow.aio_run_no_wait(input, options)

        return TaskRunRef[TWorkflowInput, R](self, ref)

    def run_many(self, workflows: list[WorkflowRunTriggerConfig]) -> list[R]:
        """
        Run a workflow in bulk and wait for all runs to complete.
        This method triggers multiple workflow runs, blocks until all of them complete, and returns the final results.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :returns: A list of results for each workflow run.
        """
        return [
            self._extract_result(result)
            for result in self._workflow.run_many(workflows)
        ]

    async def aio_run_many(self, workflows: list[WorkflowRunTriggerConfig]) -> list[R]:
        """
        Run a workflow in bulk and wait for all runs to complete.
        This method triggers multiple workflow runs, blocks until all of them complete, and returns the final results.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :returns: A list of results for each workflow run.
        """
        return [
            self._extract_result(result)
            for result in await self._workflow.aio_run_many(workflows)
        ]

    def run_many_no_wait(
        self, workflows: list[WorkflowRunTriggerConfig]
    ) -> list[TaskRunRef[TWorkflowInput, R]]:
        """
        Run a workflow in bulk without waiting for all runs to complete.

        This method triggers multiple workflow runs and immediately returns a list of references to the runs without blocking while the workflows run.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :returns: A list of `WorkflowRunRef` objects, each representing a reference to a workflow run.
        """
        refs = self._workflow.run_many_no_wait(workflows)

        return [TaskRunRef[TWorkflowInput, R](self, ref) for ref in refs]

    async def aio_run_many_no_wait(
        self, workflows: list[WorkflowRunTriggerConfig]
    ) -> list[TaskRunRef[TWorkflowInput, R]]:
        """
        Run a workflow in bulk without waiting for all runs to complete.

        This method triggers multiple workflow runs and immediately returns a list of references to the runs without blocking while the workflows run.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.

        :returns: A list of `WorkflowRunRef` objects, each representing a reference to a workflow run.
        """
        refs = await self._workflow.aio_run_many_no_wait(workflows)

        return [TaskRunRef[TWorkflowInput, R](self, ref) for ref in refs]
