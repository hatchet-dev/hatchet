from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

simple = hatchet.workflow(name="SimpleWorkflow", on_events=["simple:event"])


@simple.task(timeout="11s", retries=3)
def step1(input: EmptyModel, context: Context) -> dict[str, str]:
    print("executed step1")
    return {
        "step1": "step1",
    }


def main() -> None:
    worker = hatchet.worker("test-worker", slots=1, workflows=[simple])
    worker.start()


if __name__ == "__main__":
    main()
