from examples.affinity_workers.worker import affinity_worker_workflow

affinity_worker_workflow.run(additional_metadata={"hello": "moon"})
