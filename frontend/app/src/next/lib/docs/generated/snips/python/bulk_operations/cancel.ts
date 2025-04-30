import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': '# > Setup\n\nfrom datetime import datetime, timedelta\n\nfrom hatchet_sdk import BulkCancelReplayOpts, Hatchet, RunFilter, V1TaskStatus\n\nhatchet = Hatchet()\n\nworkflows = hatchet.workflows.list()\n\nassert workflows.rows\n\nworkflow = workflows.rows[0]\n\n\n# > List runs\nworkflow_runs = hatchet.runs.list(workflow_ids=[workflow.metadata.id])\n\n# > Cancel by run ids\nworkflow_run_ids = [workflow_run.metadata.id for workflow_run in workflow_runs.rows]\n\nbulk_cancel_by_ids = BulkCancelReplayOpts(ids=workflow_run_ids)\n\nhatchet.runs.bulk_cancel(bulk_cancel_by_ids)\n\n# > Cancel by filters\n\nbulk_cancel_by_filters = BulkCancelReplayOpts(\n    filters=RunFilter(\n        since=datetime.today() - timedelta(days=1),\n        until=datetime.now(),\n        statuses=[V1TaskStatus.RUNNING],\n        workflow_ids=[workflow.metadata.id],\n        additional_metadata={\'key\': \'value\'},\n    )\n)\n\nhatchet.runs.bulk_cancel(bulk_cancel_by_filters)\n',
  'source': 'out/python/bulk_operations/cancel.py',
  'blocks': {
    'setup': {
      'start': 2,
      'stop': 14
    },
    'list_runs': {
      'start': 17,
      'stop': 17
    },
    'cancel_by_run_ids': {
      'start': 20,
      'stop': 24
    },
    'cancel_by_filters': {
      'start': 27,
      'stop': 38
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
