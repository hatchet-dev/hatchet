import time

from examples.streaming.worker import streaming_workflow

def main() -> None:
    ref = streaming_workflow.run_no_wait()
    time.sleep(1)

    stream = ref.stream()

    for chunk in stream:
        print(chunk)

if __name__ == "__main__":
    main()
