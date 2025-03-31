from hatchet_sdk.clients.admin import AdminClient
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.features.cron import CronClient
from hatchet_sdk.features.logs import LogsClient
from hatchet_sdk.features.metrics import MetricsClient
from hatchet_sdk.features.rate_limits import RateLimitsClient
from hatchet_sdk.features.runs import RunsClient
from hatchet_sdk.features.scheduled import ScheduledClient
from hatchet_sdk.features.workers import WorkersClient
from hatchet_sdk.features.workflows import WorkflowsClient


class Client:
    def __init__(
        self,
        config: ClientConfig,
        event_client: EventClient | None = None,
        admin_client: AdminClient | None = None,
        dispatcher_client: DispatcherClient | None = None,
        workflow_listener: PooledWorkflowRunListener | None | None = None,
        debug: bool = False,
    ):
        self.config = config
        self.admin = admin_client or AdminClient(config)
        self.dispatcher = dispatcher_client or DispatcherClient(config)
        self.event = event_client or EventClient(config)
        self.listener = RunEventListenerClient(config)
        self.workflow_listener = workflow_listener
        self.log_interceptor = config.logger
        self.debug = debug

        self.cron = CronClient(self.config)
        self.logs = LogsClient(self.config)
        self.metrics = MetricsClient(self.config)
        self.rate_limits = RateLimitsClient(self.config)
        self.runs = RunsClient(self.config)
        self.scheduled = ScheduledClient(self.config)
        self.workers = WorkersClient(self.config)
        self.workflows = WorkflowsClient(self.config)
