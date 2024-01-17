from .client import new_client 
from .workflows_pb2 import CreateWorkflowVersionOpts, CreateWorkflowJobOpts, CreateWorkflowStepOpts
from dotenv import load_dotenv
from typing import Callable, List, Tuple, Any

# TODO: don't load dotenv
load_dotenv()

def workflow(name : str='', on_events : list=[], on_crons : list=[]):
   def inner(cls):
        cls.on_events = on_events
        cls.on_crons = on_crons
        cls.name = name or str(cls.__name__)

        # Define a new class with the same name and bases as the original, but with WorkflowMeta as its metaclass
        return WorkflowMeta(cls.__name__, cls.__bases__, dict(cls.__dict__))
   
   return inner

def step(name : str='', parents : List[str] = []):
    def inner(func):
        func._step_name = name or func.__name__
        func._step_parents = parents

        # print length of parents
        print("len(parents)", len(parents), len(func._step_parents))

        return func

    return inner

# def step(func):
#     if not hasattr(func, 'name'):
#         func._step_name = func.__name__
#     else:
#         func._step_name = func.name

#     if not hasattr(func, 'parents'):
#         func._step_parents = []
#     else:
#         func._step_parents = func.parents

#     return func
    

# Initialize the counter for ordering steps
step.order_counter = 0

stepsType = List[Tuple[str, Callable[..., Any]]]

class WorkflowMeta(type):
    def __new__(cls, name, bases, attrs):
        serviceName = "default"

        steps: stepsType = [(func_name, attrs.pop(func_name)) for func_name, func in list(attrs.items()) if hasattr(func, '_step_name')]

        # Define __init__ and get_step_order methods
        original_init = attrs.get('__init__')  # Get the original __init__ if it exists

        def __init__(self, *args, **kwargs):
            if original_init:
                original_init(self, *args, **kwargs)  # Call original __init__

        def get_actions(self) -> stepsType:
            return [(serviceName + ":" + func_name, func) for func_name, func in steps]
        
        # Add these methods and steps to class attributes
        attrs['__init__'] = __init__
        attrs['get_actions'] = get_actions

        for step_name, step_func in steps:
            attrs[step_name] = step_func

        # create a new hatchet client
        client = new_client()

        attrs['client'] = client

        name = attrs['name']
        event_triggers = attrs['on_events']
        cron_triggers = attrs['on_crons']

        createStepOpts: List[CreateWorkflowStepOpts] = [
            CreateWorkflowStepOpts(
                readable_id=func_name,
                action="default:" + func_name,
                timeout="60s",
                inputs='{}',
                parents=[x for x in func._step_parents]  # Assuming this is how you get the parents
            ) 
            for func_name, func in attrs.items() if hasattr(func, '_step_name')
        ]

        print("createStepOpts", createStepOpts)

        client.admin.put_workflow(CreateWorkflowVersionOpts(
            name=name,
            version="v0.62.0",
            event_triggers=event_triggers,
            cron_triggers=cron_triggers,
            jobs=[
                CreateWorkflowJobOpts(
                    name="my-job",
                    timeout="60s",
                    steps=createStepOpts,
                )
            ]
        ), auto_version=True)

        return super(WorkflowMeta, cls).__new__(cls, name, bases, attrs)
