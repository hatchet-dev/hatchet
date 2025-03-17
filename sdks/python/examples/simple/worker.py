from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

# ❓ Simple

simple = hatchet.workflow(name="SimpleWorkflow")


@simple.task()
def step1(input: EmptyModel, ctx: Context) -> None:
    print("executed step1")



def main() -> None:
    worker = hatchet.worker("test-worker", slots=1, workflows=[simple])
    worker.start()

# ‼️

if __name__ == "__main__":
    main()
