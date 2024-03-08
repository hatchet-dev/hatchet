from concurrent.futures import ThreadPoolExecutor
import datetime
import inspect
from multiprocessing import Event
import os
from .clients.dispatcher import Action, DispatcherClient
from google.protobuf import timestamp_pb2
from .clients.events import EventClientImpl
from .dispatcher_pb2 import OverridesData
from .events_pb2 import PutLogRequest
from .logger import logger
import json

def get_caller_file_path():
    caller_frame = inspect.stack()[2]

    return caller_frame.filename

class Context:
    def __init__(self, action: Action, client: DispatcherClient, eventClient: EventClientImpl):
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

        self.stepRunId = action.step_run_id
        self.exit_flag = Event()
        self.client = client
        self.eventClient = eventClient

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
        
        self.client.put_overrides_data(
            OverridesData(
                stepRunId=self.stepRunId,
                path=name,
                value=json.dumps(default),
                callerFilename=caller_file
            )
        )

        return default
    
    def _log(self, line: str):
        try:
            self.eventClient.log(message=line, step_run_id=self.stepRunId)
        except Exception as e:
            logger.error(f"Error logging: {e}")
    
    def log(self, line: str):
        if self.stepRunId == "":
            return
        
        self.logger_thread_pool.submit(self._log, line)
