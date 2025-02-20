import time

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


# ❓ Workflow Definition Cron Trigger
# Adding a cron trigger to a workflow is as simple
# as adding a `cron expression` to the `on_cron`
# prop of the workflow definition
@hatchet.workflow(on_crons=["* * * * *"])
class CronWorkflow:
    @hatchet.step()
    def step1(self, context: Context) -> dict[str, str]:

        return {
            "time": "step1",
        }


# !!


def main() -> None:
    workflow = CronWorkflow()
    worker = hatchet.worker("test-worker", max_runs=1)
    worker.register_workflow(workflow)
    worker.start()


if __name__ == "__main__":
    main()
