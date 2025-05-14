# > Simple

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

EVENT_KEY = "user:create"


@hatchet.task(name="SimpleWorkflowEventFiltering", on_events=[EVENT_KEY])
def step1(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"hello": "world!"}


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[step1])
    worker.start()


# !!

if __name__ == "__main__":
    main()
