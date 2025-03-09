import random
import time

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

wf = hatchet.workflow(name="DAGW", on_events=["dag:create"], schedule_timeout="10m")


@wf.task(timeout="5s")
def step1(input: EmptyModel, context: Context) -> dict[str, int]:
    rando = random.randint(1, 100)  # Generate a random number between 1 and 100return {
    return {
        "rando": rando,
    }


@wf.task(timeout="5s")
def step2(input: EmptyModel, context: Context) -> dict[str, int]:
    rando = random.randint(1, 100)  # Generate a random number between 1 and 100return {
    return {
        "rando": rando,
    }


@wf.task(parents=[step1, step2])
def step3(input: EmptyModel, context: Context) -> dict[str, int]:
    one = context.task_output(step1)["rando"]
    two = context.task_output(step3)["rando"]

    return {
        "sum": one + two,
    }


@wf.task(parents=[step1, step3])
def step4(input: EmptyModel, context: Context) -> dict[str, str]:
    print(
        "executed step4",
        time.strftime("%H:%M:%S", time.localtime()),
        input,
        context.task_output(step1),
        context.task_output(step3),
    )
    return {
        "step4": "step4",
    }


def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[wf])

    worker.start()


if __name__ == "__main__":
    main()
