from hatchet_sdk import Hatchet, Context
from dotenv import load_dotenv

load_dotenv()

hatchet = Hatchet(debug=True)

@hatchet.workflow(on_events=["parent:create"])
class Parent:
    @hatchet.step()
    def spawn(self, context: Context):
        print("spawning child")
        id = context.spawn_workflow("Child", key="child")
        print(f"spawned child {id}")
        pass

@hatchet.workflow(on_events=["child:create"])
class Child:
    @hatchet.step()
    def process(self, context: Context):
        print("child process")
        return {"status": "success"}


worker = hatchet.worker('fanout-worker', max_runs=4)
worker.register_workflow(Parent())
worker.register_workflow(Child())

worker.start()