package repository

type Repository interface {
	Event() EventRepository
	Tenant() TenantRepository
	Workflow() WorkflowRepository
	WorkflowRun() WorkflowRunRepository
	JobRun() JobRunRepository
	StepRun() StepRunRepository
	Step() StepRepository
	Dispatcher() DispatcherRepository
	Ticker() TickerRepository
	Worker() WorkerRepository
	UserSession() UserSessionRepository
	User() UserRepository
}

func BoolPtr(b bool) *bool {
	return &b
}

func StringPtr(s string) *string {
	return &s
}
