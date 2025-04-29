import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import time\nfrom datetime import timedelta\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet, TaskDefaults\n\nhatchet = Hatchet(debug=True)\n\n# > ScheduleTimeout\ntimeout_wf = hatchet.workflow(\n    name=\'TimeoutWorkflow\',\n    task_defaults=TaskDefaults(execution_timeout=timedelta(minutes=2)),\n)\n\n\n\n# > ExecutionTimeout\n# 👀 Specify an execution timeout on a task\n@timeout_wf.task(\n    execution_timeout=timedelta(seconds=4), schedule_timeout=timedelta(minutes=10)\n)\ndef timeout_task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    time.sleep(5)\n    return {\'status\': \'success\'}\n\n\n\n\nrefresh_timeout_wf = hatchet.workflow(name=\'RefreshTimeoutWorkflow\')\n\n\n# > RefreshTimeout\n@refresh_timeout_wf.task(execution_timeout=timedelta(seconds=4))\ndef refresh_task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n\n    ctx.refresh_timeout(timedelta(seconds=10))\n    time.sleep(5)\n\n    return {\'status\': \'success\'}\n\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'timeout-worker\', slots=4, workflows=[timeout_wf, refresh_timeout_wf]\n    )\n\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/timeout/worker.py',
  'blocks': {
    'scheduletimeout': {
      'start': 9,
      'stop': 12
    },
    'executiontimeout': {
      'start': 16,
      'stop': 24
    },
    'refreshtimeout': {
      'start': 30,
      'stop': 38
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
