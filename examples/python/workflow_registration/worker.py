# > WorkflowRegistration

from hatchet_sdk import Hatchet

hatchet = Hatchet(debug=True)

wf_one = hatchet.workflow(name="wf_one")
wf_two = hatchet.workflow(name="wf_two")
wf_three = hatchet.workflow(name="wf_three")
wf_four = hatchet.workflow(name="wf_four")
wf_five = hatchet.workflow(name="wf_five")

# define tasks here


def main() -> None:
    # 👀 Register workflows directly when instantiating the worker
    worker = hatchet.worker("test-worker", slots=1, workflows=[wf_one, wf_two])

    # 👀 Register a single workflow after instantiating the worker
    worker.register_workflow(wf_three)

    # 👀 Register multiple workflows in bulk after instantiating the worker
    worker.register_workflows([wf_four, wf_five])

    worker.start()



if __name__ == "__main__":
    main()
