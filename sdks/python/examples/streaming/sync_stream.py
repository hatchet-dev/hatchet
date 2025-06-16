import time

from examples.streaming.worker import stream_task


def main() -> None:
    ref = stream_task.run_no_wait()
    time.sleep(1)

    stream = ref._wrr.stream()

    for chunk in stream:
        print(chunk)


if __name__ == "__main__":
    main()
