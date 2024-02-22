import inspect
from multiprocessing import Event
import os
from .clients.dispatcher import Action, DispatcherClient
from .dispatcher_pb2 import OverridesData
from .logger import logger
import json

def get_caller_file_path():
    caller_frame = inspect.stack()[2]

    return caller_frame.filename

class Context:
    def __init__(self, action: Action, client: DispatcherClient):
        self.data = json.loads(action.action_payload)
        self.stepRunId = action.step_run_id
        self.exit_flag = Event()
        self.client = client

        # store each key in the overrides field in a lookup table
        # overrides_data is a dictionary of key-value pairs
        self.overrides_data = self.data.get('overrides', {})

    def step_output(self, step: str):
        try:
            return self.data['parents'][step]
        except KeyError:
            raise ValueError(f"Step output for '{step}' not found")

    def triggered_by_event(self) -> bool:
        return self.data.get('triggered_by', '') == 'event'

    def workflow_input(self):
        return self.data.get('input', {})
    
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