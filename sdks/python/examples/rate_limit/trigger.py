from examples.rate_limit.worker import rate_limit_workflow
from hatchet_sdk.hatchet import Hatchet

hatchet = Hatchet(debug=True)

rate_limit_workflow.run()
rate_limit_workflow.run()
rate_limit_workflow.run()
