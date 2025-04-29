import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import json\nfrom datetime import timedelta\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nERROR_TEXT = \'step1 failed\'\n\n# > OnFailure Step\n# This workflow will fail because the step will throw an error\n# we define an onFailure step to handle this case\n\non_failure_wf = hatchet.workflow(name=\'OnFailureWorkflow\')\n\n\n@on_failure_wf.task(execution_timeout=timedelta(seconds=1))\ndef step1(input: EmptyModel, ctx: Context) -> None:\n    # 👀 this step will always raise an exception\n    raise Exception(ERROR_TEXT)\n\n\n# 👀 After the workflow fails, this special step will run\n@on_failure_wf.on_failure_task()\ndef on_failure(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    # 👀 we can do things like perform cleanup logic\n    # or notify a user here\n\n    # 👀 Fetch the errors from upstream step runs from the context\n    print(ctx.task_run_errors)\n\n    return {\'status\': \'success\'}\n\n\n\n\n\n# > OnFailure With Details\n# We can access the failure details in the onFailure step\n# via the context method\n\non_failure_wf_with_details = hatchet.workflow(name=\'OnFailureWorkflowWithDetails\')\n\n\n# ... defined as above\n@on_failure_wf_with_details.task(execution_timeout=timedelta(seconds=1))\ndef details_step1(input: EmptyModel, ctx: Context) -> None:\n    raise Exception(ERROR_TEXT)\n\n\n# 👀 After the workflow fails, this special step will run\n@on_failure_wf_with_details.on_failure_task()\ndef details_on_failure(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    error = ctx.fetch_task_run_error(details_step1)\n\n    # 👀 we can access the failure details here\n    print(json.dumps(error, indent=2))\n\n    if error and error.startswith(ERROR_TEXT):\n        return {\'status\': \'success\'}\n\n    raise Exception(\'unexpected failure\')\n\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'on-failure-worker\',\n        slots=4,\n        workflows=[on_failure_wf, on_failure_wf_with_details],\n    )\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/on_failure/worker.py',
  'blocks': {
    'onfailure_step': {
      'start': 11,
      'stop': 34
    },
    'onfailure_with_details': {
      'start': 38,
      'stop': 63
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
