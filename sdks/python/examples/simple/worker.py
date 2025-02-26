from hatchet_sdk import BaseWorkflow, Context, Hatchet

hatchet = Hatchet(debug=True)


class MyWorkflow(BaseWorkflow):
    @hatchet.step(timeout="11s", retries=3)
    def step1(self, context: Context) -> dict[str, str]:
        print("executed step1")
        return {
            "step1": "step1",
        }


def main() -> None:
    wf = MyWorkflow()

    worker = hatchet.worker("test-worker", max_runs=1)
    worker.register_workflow(wf)
    worker.start()


if __name__ == "__main__":
    main()
