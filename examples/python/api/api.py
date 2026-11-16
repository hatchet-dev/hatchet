from hatchet_sdk import Hatchet

hatchet = Hatchet()


def main() -> None:
    workflow_list = hatchet.workflows.list()

    for workflow in workflow_list:
        print(workflow.name)
        print(workflow.metadata.id)
        print(workflow.metadata.created_at)
        print(workflow.metadata.updated_at)


if __name__ == "__main__":
    main()
