from concurrent.futures import ThreadPoolExecutor
import inspect
from multiprocessing import Event
from .clients.dispatcher import Action
from .client import ClientImpl
from .clients.admin import TriggerWorkflowParentOptions
from .clients.listener import HatchetListener, StepRunEvent, WorkflowRunEventType

from .dispatcher_pb2 import OverridesData
from .logger import logger
import json
import asyncio
from hatchet_sdk.clients.rest.models.workflow_run import WorkflowRun
from hatchet_sdk.clients.rest.models.workflow_run_status import WorkflowRunStatus
from itertools import chain
import time

DEFAULT_WORKFLOW_POLLING_INTERVAL = 5 # Seconds

def get_caller_file_path():
    caller_frame = inspect.stack()[2]

    return caller_frame.filename

class ChildWorkflowRef:
    workflow_run_id: str
    client: ClientImpl
    poll: bool = True
    
    def __init__(self, workflow_run_id: str, client: ClientImpl):
        self.workflow_run_id = workflow_run_id
        self.client = client

    def getResult(self) -> WorkflowRun:
        try:
            res: WorkflowRun = self.client.rest_client.workflow_run_get(self.workflow_run_id)
            stepRuns = res.data.jobRuns[0].stepRuns if res.data.jobRuns else []

            stepRunOutput = {}
            for stepRun in stepRuns:
                stepId = stepRun.step.readableId if stepRun.step else ''
                stepRunOutput[stepId] = json.loads(stepRun.output) if stepRun.output else {}
            
            statusMap = {
                WorkflowRunStatus.SUCCEEDED: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_COMPLETED,
                WorkflowRunStatus.FAILED: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FAILED,
                WorkflowRunStatus.CANCELLED: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_CANCELLED,
            }

            if res.data.status in statusMap:
                return {
                    'type': statusMap[res.data.status],
                    'payload': json.dumps(stepRunOutput)
                }

        except Exception as e:
            # TODO
            raise Exception(str(e))

    def polling(self):
        while self.poll:
            res = self.getResult()
            if res:
                yield self.handle_event(res)
            print('polling')
            time.sleep(DEFAULT_WORKFLOW_POLLING_INTERVAL)

    def stream(self):
        listener_stream = self.client.listener.stream(self.workflow_run_id)
        self.poll = True
        return chain(self.polling(), listener_stream)

    async def result(self):
        try:
            for event in self.stream():
                print(event.type)
                self.handle_event(event)
        finally:
            self.close()
            print('listener closed')

    def close(self):
        self.poll = False

    def handle_event(self, event: StepRunEvent):
        if (
            event.type == WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FAILED
            or event.type == WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_CANCELLED
            or event.type == WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_TIMED_OUT
        ): 
            self.close()
            raise RuntimeError(event.type)

        if event.type == WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_COMPLETED:
            self.close()
            return json.loads(event.payload)


class Context:
    spawn_index = -1

    def __init__(self, action: Action, client: ClientImpl):
        # Check the type of action.action_payload before attempting to load it as JSON
        if isinstance(action.action_payload, (str, bytes, bytearray)):
            try:
                self.data = json.loads(action.action_payload)
            except Exception as e:
                logger.error(f"Error parsing action payload: {e}")
                # Assign an empty dictionary if parsing fails
                self.data = {}
        else:
            # Directly assign the payload to self.data if it's already a dict
            self.data = action.action_payload if isinstance(action.action_payload, dict) else {}

        self.action = action
        self.stepRunId = action.step_run_id
        self.exit_flag = Event()
        self.client = client

        # FIXME: this limits the number of concurrent log requests to 1, which means we can do about
        # 100 log lines per second but this depends on network. 
        self.logger_thread_pool = ThreadPoolExecutor(max_workers=1)

        # store each key in the overrides field in a lookup table
        # overrides_data is a dictionary of key-value pairs
        self.overrides_data = self.data.get('overrides', {})

        if action.get_group_key_run_id != "":
            self.input = self.data
        else:
            self.input = self.data.get('input', {})

    def step_output(self, step: str):
        try:
            return self.data['parents'][step]
        except KeyError:
            raise ValueError(f"Step output for '{step}' not found")

    def triggered_by_event(self) -> bool:
        return self.data.get('triggered_by', '') == 'event'

    def workflow_input(self):
        return self.input
    
    def workflow_run_id(self):
        return self.action.workflow_run_id

    def sleep(self, seconds: int):
        self.exit_flag.wait(seconds)

        if self.exit_flag.is_set():
            raise Exception("Context cancelled")
    
    def cancel(self):
        logger.info("Cancelling step...")
        self.exit_flag.set()

    # done returns true if the context has been cancelled
    def done(self):
        return self.exit_flag.is_set()
    
    def playground(self, name: str, default: str = None):
        # if the key exists in the overrides_data field, return the value
        if name in self.overrides_data:
            return self.overrides_data[name]
        
        caller_file = get_caller_file_path()
        
        self.client.dispatcher.put_overrides_data(
            OverridesData(
                stepRunId=self.stepRunId,
                path=name,
                value=json.dumps(default),
                callerFilename=caller_file
            )
        )

        return default
    
    def spawn_workflow(self, workflow_name: str, input: dict = {}, key: str = None):
        workflow_run_id = self.action.workflow_run_id
        step_run_id = self.action.step_run_id

        print(f"Spawning workflow {workflow_run_id}")
        print(f"Spawning workflow {step_run_id}")

        options: TriggerWorkflowParentOptions = {
            'parent_id': workflow_run_id,
            'parent_step_run_id': step_run_id,
            'child_key': key,
            'child_index': self.spawn_index
        }

        self.spawn_index += 1

        child_workflow_run_id = self.client.admin.run_workflow(
            workflow_name,
            input,
            options
        )

        return ChildWorkflowRef(child_workflow_run_id, self.client)

    def _log(self, line: str):
        try:
            self.client.event.log(message=line, step_run_id=self.stepRunId)
        except Exception as e:
            logger.error(f"Error logging: {e}")
    
    def log(self, line: str):
        if self.stepRunId == "":
            return
        
        self.logger_thread_pool.submit(self._log, line)
