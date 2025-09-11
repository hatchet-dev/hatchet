from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


@hatchet.task()
def bulk_replay_test_1(input: EmptyModel, ctx: Context) -> None:
    print("retrying bulk replay test task", ctx.retry_count)
    if ctx.retry_count == 0:
        raise ValueError("This is a test error to trigger a retry.")


@hatchet.task()
def bulk_replay_test_2(input: EmptyModel, ctx: Context) -> None:
    print("retrying bulk replay test task", ctx.retry_count)
    if ctx.retry_count == 0:
        raise ValueError("This is a test error to trigger a retry.")


@hatchet.task()
def bulk_replay_test_3(input: EmptyModel, ctx: Context) -> None:
    print("retrying bulk replay test task", ctx.retry_count)
    if ctx.retry_count == 0:
        raise ValueError("This is a test error to trigger a retry.")


def main() -> None:
    worker = hatchet.worker(
        "bulk-replay-test-worker",
        workflows=[bulk_replay_test_1, bulk_replay_test_2, bulk_replay_test_3],
    )

    worker.start()


if __name__ == "__main__":
    main()
