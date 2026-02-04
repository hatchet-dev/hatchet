from examples.concurrency_limit.worker import WorkflowInput, concurrency_limit_workflow

concurrency_limit_workflow.run(WorkflowInput(group_key="test", run=1))
