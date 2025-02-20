from hatchet_sdk import BaseWorkflow, Context, Hatchet

hatchet = Hatchet(debug=True)


# ❓ Workflow Definition Cron Trigger
# Adding a cron trigger to a workflow is as simple
# as adding a `cron expression` to the `on_cron`
# prop of the workflow definition

wf = hatchet.declare_workflow(on_crons=["* * * * *"])


class CronWorkflow(BaseWorkflow):
    config = wf.config

    @hatchet.step()
    def step1(self, context: Context) -> dict[str, str]:

        return {
            "time": "step1",
        }


# !!


def main() -> None:
    worker = hatchet.worker("test-worker", max_runs=1)
    worker.register_workflow(CronWorkflow())
    worker.start()


if __name__ == "__main__":
    main()
