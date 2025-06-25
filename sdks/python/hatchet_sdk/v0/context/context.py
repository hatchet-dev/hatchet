import inspect
import json
import traceback
from concurrent.futures import Future, ThreadPoolExecutor
from typing import Any, Generic, Type, TypeVar, cast, overload
from warnings import warn

from pydantic import BaseModel, StrictStr

from hatchet_sdk.contracts.dispatcher_pb2 import OverridesData
from hatchet_sdk.contracts.workflows_pb2 import (
    BulkTriggerWorkflowRequest,
    TriggerWorkflowRequest,
)
from hatchet_sdk.v0.clients.events import EventClient
from hatchet_sdk.v0.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.v0.clients.rest_client import RestApi
from hatchet_sdk.v0.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.v0.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.v0.context.worker_context import WorkerContext
from hatchet_sdk.v0.utils.types import WorkflowValidator
from hatchet_sdk.v0.utils.typing import is_basemodel_subclass
from hatchet_sdk.v0.workflow_run import WorkflowRunRef

from ...logger import logger
from ..clients.admin import (
    AdminClient,
    ChildTriggerWorkflowOptions,
    ChildWorkflowRunDict,
    TriggerWorkflowOptions,
    WorkflowRunDict,
)
from ..clients.dispatcher.dispatcher import (  # type: ignore[attr-defined]
    Action,
    DispatcherClient,
)

DEFAULT_WORKFLOW_POLLING_INTERVAL = 5  # Seconds

T = TypeVar("T", bound=BaseModel)


def get_caller_file_path() -> str:
    caller_frame = inspect.stack()[2]

    return caller_frame.filename


class BaseContext:

    action: Action
    spawn_index: int

    def _prepare_workflow_options(
        self,
        key: str | None = None,
        options: ChildTriggerWorkflowOptions | None = None,
        worker_id: str | None = None,
    ) -> TriggerWorkflowOptions:
        workflow_run_id = self.action.workflow_run_id
        step_run_id = self.action.step_run_id

        desired_worker_id = None
        if options is not None and "sticky" in options and options["sticky"] == True:
            desired_worker_id = worker_id

        meta = None
        if options is not None and "additional_metadata" in options:
            meta = options["additional_metadata"]

        ## TODO: Pydantic here to simplify this
        trigger_options: TriggerWorkflowOptions = {
            "parent_id": workflow_run_id,
            "parent_step_run_id": step_run_id,
            "child_key": key,
            "child_index": self.spawn_index,
            "additional_metadata": meta,
            "desired_worker_id": desired_worker_id,
        }

        self.spawn_index += 1
        return trigger_options


class ContextAioImpl(BaseContext):
    def __init__(
        self,
        action: Action,
        dispatcher_client: DispatcherClient,
        admin_client: AdminClient,
        event_client: EventClient,
        rest_client: RestApi,
        workflow_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
        worker: WorkerContext,
        namespace: str = "",
    ):
        self.action = action
        self.dispatcher_client = dispatcher_client
        self.admin_client = admin_client
        self.event_client = event_client
        self.rest_client = rest_client
        self.workflow_listener = workflow_listener
        self.workflow_run_event_listener = workflow_run_event_listener
        self.namespace = namespace
        self.spawn_index = -1
        self.worker = worker

    @tenacity_retry
    async def spawn_workflow(
        self,
        workflow_name: str,
        input: dict[str, Any] = {},
        key: str | None = None,
        options: ChildTriggerWorkflowOptions | None = None,
    ) -> WorkflowRunRef:
        worker_id = self.worker.id()
        # if (
        #     options is not None
        #     and "sticky" in options
        #     and options["sticky"] == True
        #     and not self.worker.has_workflow(workflow_name)
        # ):
        #     raise Exception(
        #         f"cannot run with sticky: workflow {workflow_name} is not registered on the worker"
        #     )

        trigger_options = self._prepare_workflow_options(key, options, worker_id)

        return await self.admin_client.aio.run_workflow(
            workflow_name, input, trigger_options
        )

    @tenacity_retry
    async def spawn_workflows(
        self, child_workflow_runs: list[ChildWorkflowRunDict]
    ) -> list[WorkflowRunRef]:

        if len(child_workflow_runs) == 0:
            raise Exception("no child workflows to spawn")

        worker_id = self.worker.id()

        bulk_trigger_workflow_runs: list[WorkflowRunDict] = []
        for child_workflow_run in child_workflow_runs:
            workflow_name = child_workflow_run["workflow_name"]
            input = child_workflow_run["input"]

            key = child_workflow_run.get("key")
            options = child_workflow_run.get("options", {})

            trigger_options = self._prepare_workflow_options(key, options, worker_id)

            bulk_trigger_workflow_runs.append(
                WorkflowRunDict(
                    workflow_name=workflow_name, input=input, options=trigger_options
                )
            )

        return await self.admin_client.aio.run_workflows(bulk_trigger_workflow_runs)


class Context(BaseContext):
    spawn_index = -1

    worker: WorkerContext

    def __init__(
        self,
        action: Action,
        dispatcher_client: DispatcherClient,
        admin_client: AdminClient,
        event_client: EventClient,
        rest_client: RestApi,
        workflow_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
        worker: WorkerContext,
        namespace: str = "",
        validator_registry: dict[str, WorkflowValidator] = {},
    ):
        self.worker = worker
        self.validator_registry = validator_registry

        self.aio = ContextAioImpl(
            action,
            dispatcher_client,
            admin_client,
            event_client,
            rest_client,
            workflow_listener,
            workflow_run_event_listener,
            worker,
            namespace,
        )

        # Check the type of action.action_payload before attempting to load it as JSON
        if isinstance(action.action_payload, (str, bytes, bytearray)):
            try:
                self.data = cast(dict[str, Any], json.loads(action.action_payload))
            except Exception as e:
                logger.error(f"Error parsing action payload: {e}")
                # Assign an empty dictionary if parsing fails
                self.data: dict[str, Any] = {}  # type: ignore[no-redef]
        else:
            # Directly assign the payload to self.data if it's already a dict
            self.data = (
                action.action_payload if isinstance(action.action_payload, dict) else {}
            )

        self.action = action

        # FIXME: stepRunId is a legacy field, we should remove it
        self.stepRunId = action.step_run_id

        self.step_run_id = action.step_run_id
        self.exit_flag = False
        self.dispatcher_client = dispatcher_client
        self.admin_client = admin_client
        self.event_client = event_client
        self.rest_client = rest_client
        self.workflow_listener = workflow_listener
        self.workflow_run_event_listener = workflow_run_event_listener
        self.namespace = namespace

        # FIXME: this limits the number of concurrent log requests to 1, which means we can do about
        # 100 log lines per second but this depends on network.
        self.logger_thread_pool = ThreadPoolExecutor(max_workers=1)
        self.stream_event_thread_pool = ThreadPoolExecutor(max_workers=1)

        # store each key in the overrides field in a lookup table
        # overrides_data is a dictionary of key-value pairs
        self.overrides_data = self.data.get("overrides", {})

        if action.get_group_key_run_id != "":
            self.input = self.data
        else:
            self.input = self.data.get("input", {})

    def step_output(self, step: str) -> dict[str, Any] | BaseModel:
        workflow_validator = next(
            (v for k, v in self.validator_registry.items() if k.split(":")[-1] == step),
            None,
        )

        try:
            parent_step_data = cast(dict[str, Any], self.data["parents"][step])
        except KeyError:
            raise ValueError(f"Step output for '{step}' not found")

        if workflow_validator and (v := workflow_validator.step_output):
            return v.model_validate(parent_step_data)

        return parent_step_data

    def triggered_by_event(self) -> bool:
        return cast(str, self.data.get("triggered_by", "")) == "event"

    def workflow_input(self) -> dict[str, Any] | T:
        if (r := self.validator_registry.get(self.action.action_id)) and (
            i := r.workflow_input
        ):
            return cast(
                T,
                i.model_validate(self.input),
            )

        return self.input

    def workflow_run_id(self) -> str:
        return self.action.workflow_run_id

    def cancel(self) -> None:
        logger.debug("cancelling step...")
        self.exit_flag = True

    # done returns true if the context has been cancelled
    def done(self) -> bool:
        return self.exit_flag

    def playground(self, name: str, default: str | None = None) -> str | None:
        # if the key exists in the overrides_data field, return the value
        if name in self.overrides_data:
            warn(
                "Use of `overrides_data` is deprecated.",
                DeprecationWarning,
                stacklevel=1,
            )
            return str(self.overrides_data[name])

        caller_file = get_caller_file_path()

        self.dispatcher_client.put_overrides_data(
            OverridesData(
                stepRunId=self.stepRunId,
                path=name,
                value=json.dumps(default),
                callerFilename=caller_file,
            )
        )

        return default

    def _log(self, line: str) -> tuple[bool, Exception | None]:
        try:
            self.event_client.log(message=line, step_run_id=self.stepRunId)
            return True, None
        except Exception as e:
            # we don't want to raise an exception here, as it will kill the log thread
            return False, e

    def log(self, line: Any, raise_on_error: bool = False) -> None:
        if self.stepRunId == "":
            return

        if not isinstance(line, str):
            try:
                line = json.dumps(line)
            except Exception:
                line = str(line)

        future = self.logger_thread_pool.submit(self._log, line)

        def handle_result(future: Future[tuple[bool, Exception | None]]) -> None:
            success, exception = future.result()
            if not success and exception:
                if raise_on_error:
                    raise exception
                else:
                    thread_trace = "".join(
                        traceback.format_exception(
                            type(exception), exception, exception.__traceback__
                        )
                    )
                    call_site_trace = "".join(traceback.format_stack())
                    logger.error(
                        f"Error in log thread: {exception}\n{thread_trace}\nCalled from:\n{call_site_trace}"
                    )

        future.add_done_callback(handle_result)

    def release_slot(self) -> None:
        return self.dispatcher_client.release_slot(self.stepRunId)

    def _put_stream(self, data: str | bytes) -> None:
        try:
            self.event_client.stream(data=data, step_run_id=self.stepRunId)
        except Exception as e:
            logger.error(f"Error putting stream event: {e}")

    def put_stream(self, data: str | bytes) -> None:
        if self.stepRunId == "":
            return

        self.stream_event_thread_pool.submit(self._put_stream, data)

    def refresh_timeout(self, increment_by: str) -> None:
        try:
            return self.dispatcher_client.refresh_timeout(
                step_run_id=self.stepRunId, increment_by=increment_by
            )
        except Exception as e:
            logger.error(f"Error refreshing timeout: {e}")

    def retry_count(self) -> int:
        return self.action.retry_count

    def additional_metadata(self) -> dict[str, Any] | None:
        return self.action.additional_metadata

    def child_index(self) -> int | None:
        return self.action.child_workflow_index

    def child_key(self) -> str | None:
        return self.action.child_workflow_key

    def parent_workflow_run_id(self) -> str | None:
        return self.action.parent_workflow_run_id

    def step_run_errors(self) -> dict[str, str]:
        errors = cast(dict[str, str], self.data.get("step_run_errors", {}))

        if not errors:
            logger.error(
                "No step run errors found. `context.step_run_errors` is intended to be run in an on-failure step, and will only work on engine versions more recent than v0.53.10"
            )

        return errors

    def fetch_run_failures(self) -> list[dict[str, StrictStr]]:
        data = self.rest_client.workflow_run_get(self.action.workflow_run_id)
        other_job_runs = [
            run for run in (data.job_runs or []) if run.job_id != self.action.job_id
        ]
        # TODO: Parse Step Runs using a Pydantic Model rather than a hand crafted dictionary
        return [
            {
                "step_id": step_run.step_id,
                "step_run_action_name": step_run.step.action,
                "error": step_run.error,
            }
            for job_run in other_job_runs
            if job_run.step_runs
            for step_run in job_run.step_runs
            if step_run.error and step_run.step
        ]

    @tenacity_retry
    def spawn_workflow(
        self,
        workflow_name: str,
        input: dict[str, Any] = {},
        key: str | None = None,
        options: ChildTriggerWorkflowOptions | None = None,
    ) -> WorkflowRunRef:
        worker_id = self.worker.id()
        trigger_options = self._prepare_workflow_options(key, options, worker_id)

        return self.admin_client.run_workflow(workflow_name, input, trigger_options)

    @tenacity_retry
    def spawn_workflows(
        self, child_workflow_runs: list[ChildWorkflowRunDict]
    ) -> list[WorkflowRunRef]:

        if len(child_workflow_runs) == 0:
            raise Exception("no child workflows to spawn")

        worker_id = self.worker.id()

        bulk_trigger_workflow_runs: list[WorkflowRunDict] = []
        for child_workflow_run in child_workflow_runs:
            workflow_name = child_workflow_run["workflow_name"]
            input = child_workflow_run["input"]

            key = child_workflow_run.get("key")
            options = child_workflow_run.get("options", {})

            trigger_options = self._prepare_workflow_options(key, options, worker_id)

            bulk_trigger_workflow_runs.append(
                WorkflowRunDict(
                    workflow_name=workflow_name, input=input, options=trigger_options
                )
            )

        return self.admin_client.run_workflows(bulk_trigger_workflow_runs)
