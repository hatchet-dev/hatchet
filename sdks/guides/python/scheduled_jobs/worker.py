from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()


# > Step 01 Define Cron Task
cron_wf = hatchet.workflow(name="ScheduledWorkflow", on_crons=["0 * * * *"])


@cron_wf.task()
def run_scheduled_job(input: EmptyModel, ctx: Context) -> dict:
    """Runs every hour (minute 0)."""
    return {"status": "completed", "job": "maintenance"}


# !!


def main() -> None:
    # > Step 03 Run Worker
    worker = hatchet.worker(
        "scheduled-worker",
        workflows=[cron_wf],
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
