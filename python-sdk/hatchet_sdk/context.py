from multiprocessing import Event
from .logger import logger
import json

class Context:
    def __init__(self, payload: str):
        self.data = json.loads(payload)
        self.exit_flag = Event()

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
