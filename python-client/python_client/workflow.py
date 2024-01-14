def workflow(cls):
    # Define a new class with the same name and bases as the original, but with WorkflowMeta as its metaclass
    return WorkflowMeta(cls.__name__, cls.__bases__, dict(cls.__dict__))

def step(func):
    if not hasattr(func, '_order'):
        func._order = step.order_counter
        step.order_counter += 1
    return func

# Initialize the counter for ordering steps
step.order_counter = 0

class WorkflowMeta(type):
    def __new__(cls, name, bases, attrs):
        # Collect steps and sort them based on their order
        steps = [(func_name, attrs.pop(func_name)) for func_name, func in list(attrs.items()) if hasattr(func, '_order')]
        steps.sort(key=lambda x: x[1]._order)

        # Define __init__ and get_step_order methods
        original_init = attrs.get('__init__')  # Get the original __init__ if it exists

        def __init__(self, *args, **kwargs):
            if original_init:
                original_init(self, *args, **kwargs)  # Call original __init__
            self._step_order = [step[0] for step in steps]

        def get_step_order(self):
            return self._step_order
        
        def invoke_step(self, step_name, context):
            return getattr(self, step_name)(context)

        # Add these methods and steps to class attributes
        attrs['__init__'] = __init__
        attrs['get_step_order'] = get_step_order
        attrs['invoke_step'] = invoke_step
        for step_name, step_func in steps:
            attrs[step_name] = step_func

        return super(WorkflowMeta, cls).__new__(cls, name, bases, attrs)

# EXAMPLE - MOVE TO EXAMPLES DIRECTORY
@workflow
class MyWorkflow:
    def __init__(self):
        # MyWorkflow specific initialization
        print("MyWorkflow initialization")
        self.my_value = "test"

    @step
    def step1(self, context):
        print("MyWorkflow step1", self.my_value)
        pass

    @step
    def step2(self, context):
        pass

workflow = MyWorkflow()
print(workflow.get_step_order())
workflow.invoke_step('step1', None)
