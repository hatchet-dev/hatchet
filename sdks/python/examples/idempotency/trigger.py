from examples.idempotency.worker import idempotent_task, IdempotencyInput

idempotent_task.run(input=IdempotencyInput(id=123), wait_for_result=False)
idempotent_task.run(input=IdempotencyInput(id=123), wait_for_result=False)
