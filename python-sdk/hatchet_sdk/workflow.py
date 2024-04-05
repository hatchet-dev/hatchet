from .workflows_pb2 import CreateWorkflowVersionOpts, CreateWorkflowJobOpts, CreateWorkflowStepOpts, WorkflowConcurrencyOpts
from typing import Callable, List, Tuple, Any

stepsType = List[Tuple[str, Callable[..., Any]]]

class WorkflowMeta(type):
    def __new__(cls, name, bases, attrs):

        namespace = attrs['client'].config.namespace

        serviceName = namespace + name.lower()

        concurrencyActions: stepsType = [(func_name, attrs.pop(func_name)) for func_name, func in list(attrs.items()) if hasattr(func, '_concurrency_fn_name')]
        steps: stepsType = [(func_name, attrs.pop(func_name)) for func_name, func in list(attrs.items()) if hasattr(func, '_step_name')]

        # Define __init__ and get_step_order methods
        original_init = attrs.get('__init__')  # Get the original __init__ if it exists

        def __init__(self, *args, **kwargs):
            if original_init:
                original_init(self, *args, **kwargs)  # Call original __init__

        def get_actions(self) -> stepsType:
            func_actions = [(serviceName + ":" + func_name, func) for func_name, func in steps]
            concurrency_actions = [(serviceName + ":" + func_name, func) for func_name, func in concurrencyActions]

            return func_actions + concurrency_actions
        
        # Add these methods and steps to class attributes
        attrs['__init__'] = __init__
        attrs['get_actions'] = get_actions

        for step_name, step_func in steps:
            attrs[step_name] = step_func

        name = namespace + attrs['name']
        event_triggers = [namespace + event for event in attrs['on_events']]
        cron_triggers = attrs['on_crons']
        version = attrs['version']
        workflowTimeout = attrs['timeout']
        schedule_timeout = attrs['schedule_timeout']

        createStepOpts: List[CreateWorkflowStepOpts] = [
            CreateWorkflowStepOpts(
                readable_id=func_name,
                action=serviceName + ":" + func_name,
                timeout=func._step_timeout or "60s",
                inputs='{}',
                parents=[x for x in func._step_parents],
                retries=func._step_retries,
                rate_limits=func._step_rate_limits,
            )
            for func_name, func in attrs.items() if hasattr(func, '_step_name')
        ]

        concurrency : WorkflowConcurrencyOpts | None = None

        if len(concurrencyActions) > 0:
            action = concurrencyActions[0]

            concurrency = WorkflowConcurrencyOpts(
                action=serviceName + ":" + action[0],
                max_runs=action[1]._concurrency_max_runs,
                limit_strategy=action[1]._concurrency_limit_strategy,
            )

        def get_create_opts(self):
            return CreateWorkflowVersionOpts(
                name=name,
                version=version,
                event_triggers=event_triggers,
                cron_triggers=cron_triggers,
                schedule_timeout=schedule_timeout,
                jobs=[
                    CreateWorkflowJobOpts(
                        name=name,
                        timeout=workflowTimeout,
                        steps=createStepOpts,
                    )
                ],
                concurrency=concurrency,
            )
    
        def get_name(self):
            return name
        
        attrs['get_create_opts'] = get_create_opts
        attrs['get_name'] = get_name

        return super(WorkflowMeta, cls).__new__(cls, name, bases, attrs)
