from .client import new_client 
from typing import List
from .workflow import WorkflowMeta
from .worker import Worker
from .logger import logger

class Hatchet:
    def __init__(self, debug=False):
        # initialize a client
        self.client = new_client()

        if not debug:
            logger.disable("hatchet_sdk")

    def concurrency(self, name : str='', max_runs : int = 1):
        def inner(func):
            func._concurrency_fn_name = name or func.__name__
            func._concurrency_max_runs = max_runs

            return func
        
        return inner

    def workflow(self, name : str='', on_events : list=[], on_crons : list=[], version : str='', timeout : str = '60m'):
        def inner(cls):
                cls.on_events = on_events
                cls.on_crons = on_crons
                cls.name = name or str(cls.__name__)
                cls.client = self.client
                cls.version = version
                cls.timeout = timeout

                # Define a new class with the same name and bases as the original, but with WorkflowMeta as its metaclass
                return WorkflowMeta(cls.__name__, cls.__bases__, dict(cls.__dict__))
        
        return inner

    def step(self, name : str='', timeout : str='', parents : List[str] = []):
        def inner(func):
            func._step_name = name or func.__name__
            func._step_parents = parents
            func._step_timeout = timeout

            return func

        return inner
    
    def worker(self, name: str, max_threads: int = 200):
        return Worker(name, max_threads)
