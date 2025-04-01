# ❓ Simple

from hatchet_sdk import Context, Hatchet
from pydantic import BaseModel

hatchet = Hatchet(debug=True)

class SimpleInput(BaseModel):
    message: str

@hatchet.task(name="SimpleWorkflow")
def step1(input: SimpleInput, ctx: Context) -> None:
    print("executed step1: ", input.message)


def main() -> None:
    worker = hatchet.worker("test-worker", slots=1, workflows=[step1])
    worker.start()


# ‼️

if __name__ == "__main__":
    main()
