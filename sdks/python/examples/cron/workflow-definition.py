from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


# ❓ Workflow Definition Cron Trigger
# Adding a cron trigger to a workflow is as simple
# as adding a `cron expression` to the `on_cron`
# prop of the workflow definition

wf = hatchet.workflow(name="CronWorkflow", on_crons=["* * * * *"])


@wf.task()
def step1(input: EmptyModel, context: Context) -> dict[str, str]:
    return {
        "time": "step1",
    }


# !!


def main() -> None:
    worker = hatchet.worker("test-worker", max_runs=1, workflows=[wf])
    worker.start()


if __name__ == "__main__":
    main()
