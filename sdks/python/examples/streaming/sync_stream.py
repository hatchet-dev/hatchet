from examples.streaming.worker import streaming_workflow


def main() -> None:
    ref = streaming_workflow.run_no_wait()

    stream = ref.stream()

    for chunk in stream:
        print(chunk)


if __name__ == "__main__":
    main()
