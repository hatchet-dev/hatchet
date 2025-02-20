from examples.v2.workflows import ExampleWorkflowInput, example_workflow


def main() -> None:
    example_workflow.run(
        input=ExampleWorkflowInput(message="Hello, world!"),
    )


if __name__ == "__main__":
    main()
