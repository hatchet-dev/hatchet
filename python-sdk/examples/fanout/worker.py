from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["parent:create"])
class Parent:
    @hatchet.step(timeout="10s")
    async def spawn(self, context: Context):
        print("spawning child")
        child = await context.spawn_workflow("Child", key="child").result()
        print(f"results {child}")


@hatchet.workflow(on_events=["child:create"])
class Child:
    @hatchet.step()
    def process(self, context: Context):
        print("child process")
        return {"status": "success"}


worker = hatchet.worker("fanout-worker", max_runs=4)
worker.register_workflow(Parent())
worker.register_workflow(Child())

worker.start()
