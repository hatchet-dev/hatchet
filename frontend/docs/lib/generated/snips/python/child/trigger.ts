import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "# ruff: noqa: E402\n\nimport asyncio\n\n# > Running a Task\nfrom examples.child.worker import SimpleInput, child_task\n\nchild_task.run(SimpleInput(message=\"Hello, World!\"))\n\n# > Schedule a Task\nfrom datetime import datetime, timedelta, timezone\n\nchild_task.schedule(\n    datetime.now(tz=timezone.utc) + timedelta(minutes=5),\n    SimpleInput(message=\"Hello, World!\"),\n)\n\n\nasync def main() -> None:\n    # > Running a Task AIO\n    result = await child_task.aio_run(SimpleInput(message=\"Hello, World!\"))\n\n    print(result)\n\n    # > Running Multiple Tasks\n    result1 = child_task.aio_run(SimpleInput(message=\"Hello, World!\"))\n    result2 = child_task.aio_run(SimpleInput(message=\"Hello, Moon!\"))\n\n    #  gather the results of the two tasks\n    results = await asyncio.gather(result1, result2)\n\n    #  print the results of the two tasks\n    print(results[0][\"transformed_message\"])\n    print(results[1][\"transformed_message\"])\n",
  "source": "out/python/child/trigger.py",
  "blocks": {
    "running_a_task": {
      "start": 6,
      "stop": 8
    },
    "schedule_a_task": {
      "start": 11,
      "stop": 16
    },
    "running_a_task_aio": {
      "start": 21,
      "stop": 21
    },
    "running_multiple_tasks": {
      "start": 26,
      "stop": 34
    }
  },
  "highlights": {}
};

export default snippet;
