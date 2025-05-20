import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from datetime import datetime, timedelta\n\nfrom examples.priority.worker import priority_workflow\nfrom hatchet_sdk import ScheduleTriggerWorkflowOptions, TriggerWorkflowOptions\n\npriority_workflow.run_no_wait()\n\n# > Runtime priority\nlow_prio = priority_workflow.run_no_wait(\n    options=TriggerWorkflowOptions(\n        ## 👀 Adding priority and key to metadata to show them in the dashboard\n        priority=1,\n        additional_metadata={\"priority\": \"low\", \"key\": 1},\n    )\n)\n\nhigh_prio = priority_workflow.run_no_wait(\n    options=TriggerWorkflowOptions(\n        ## 👀 Adding priority and key to metadata to show them in the dashboard\n        priority=3,\n        additional_metadata={\"priority\": \"high\", \"key\": 1},\n    )\n)\n\n# > Scheduled priority\nschedule = priority_workflow.schedule(\n    run_at=datetime.now() + timedelta(minutes=1),\n    options=ScheduleTriggerWorkflowOptions(priority=3),\n)\n\ncron = priority_workflow.create_cron(\n    cron_name=\"my-scheduled-cron\",\n    expression=\"0 * * * *\",\n    priority=3,\n)\n\n# > Default priority\nlow_prio = priority_workflow.run_no_wait(\n    options=TriggerWorkflowOptions(\n        ## 👀 Adding priority and key to metadata to show them in the dashboard\n        priority=1,\n        additional_metadata={\"priority\": \"low\", \"key\": 2},\n    )\n)\nhigh_prio = priority_workflow.run_no_wait(\n    options=TriggerWorkflowOptions(\n        ## 👀 Adding priority and key to metadata to show them in the dashboard\n        priority=3,\n        additional_metadata={\"priority\": \"high\", \"key\": 2},\n    )\n)\n",
  "source": "out/python/priority/trigger.py",
  "blocks": {
    "runtime_priority": {
      "start": 9,
      "stop": 23
    },
    "scheduled_priority": {
      "start": 26,
      "stop": 35
    }
  },
  "highlights": {}
};

export default snippet;
