from examples.dataclasses.worker import Input, say_hello

# > Triggering a task
say_hello.run(input=Input(name="Hatchet"))
