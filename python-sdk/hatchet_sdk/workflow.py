from .client import new_client 
from .workflows_pb2 import CreateWorkflowVersionOpts, CreateWorkflowJobOpts, CreateWorkflowStepOpts
from typing import Callable, List, Tuple, Any

stepsType = List[Tuple[str, Callable[..., Any]]]

class WorkflowMeta(type):
    def __new__(cls, name, bases, attrs):
        serviceName = "default"

        steps: stepsType = [(name.lower() + "-" + func_name, attrs.pop(func_name)) for func_name, func in list(attrs.items()) if hasattr(func, '_step_name')]

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
        client = attrs['client'] if 'client' in attrs else new_client()

        attrs['client'] = client

        name = attrs['name']
        event_triggers = attrs['on_events']
        cron_triggers = attrs['on_crons']
        version = attrs['version']

        createStepOpts: List[CreateWorkflowStepOpts] = [
            CreateWorkflowStepOpts(
                readable_id=func_name,
                action="default:" + func_name,
                timeout=func._step_timeout or "60s",
                inputs='{}',
                parents=[x for x in func._step_parents]  # Assuming this is how you get the parents
            ) 
            for func_name, func in attrs.items() if hasattr(func, '_step_name')
        ]

        client.admin.put_workflow(CreateWorkflowVersionOpts(
            name=name,
            version=version,
            event_triggers=event_triggers,
            cron_triggers=cron_triggers,
            jobs=[
                CreateWorkflowJobOpts(
                    name="my-job",
                    timeout="60s",
                    steps=createStepOpts,
                )
            ]
        ))

        return super(WorkflowMeta, cls).__new__(cls, name, bases, attrs)
