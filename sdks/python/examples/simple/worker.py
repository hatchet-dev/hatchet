# > Simple

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


@hatchet.task(name="SimpleWorkflow")
async def step1(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print("executed step1")
    return {
        "step1": "step1",
    }


x = step1.run()


def main() -> None:
    worker = hatchet.worker("test-worker", slots=1, workflows=[step1])
    worker.start()


# !!

if __name__ == "__main__":
    main()
