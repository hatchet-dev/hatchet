import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    '# > LoggingWorkflow\n\nimport logging\nimport time\n\nfrom examples.logger.client import hatchet\nfrom hatchet_sdk import Context, EmptyModel\n\nlogger = logging.getLogger(__name__)\n\nlogging_workflow = hatchet.workflow(\n    name="LoggingWorkflow",\n)\n\n\n@logging_workflow.task()\ndef root_logger(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(12):\n        logger.info(f"executed step1 - {i}")\n        logger.info({"step1": "step1"})\n\n        time.sleep(0.1)\n\n    return {"status": "success"}\n\n\n\n# > ContextLogger\n\n\n@logging_workflow.task()\ndef context_logger(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(12):\n        ctx.log(f"executed step1 - {i}")\n        ctx.log({"step1": "step1"})\n\n        time.sleep(0.1)\n\n    return {"status": "success"}\n\n\n',
  source: 'out/python/logger/workflow.py',
  blocks: {
    loggingworkflow: {
      start: 2,
      stop: 26,
    },
    contextlogger: {
      start: 29,
      stop: 41,
    },
  },
  highlights: {},
};

export default snippet;
