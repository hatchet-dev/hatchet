package repository

type Repository interface {
	Health() HealthRepository
	APIToken() APITokenRepository
	Event() EventRepository
	Tenant() TenantRepository
	TenantInvite() TenantInviteRepository
	Workflow() WorkflowRepository
	WorkflowRun() WorkflowRunRepository
	JobRun() JobRunRepository
	StepRun() StepRunRepository
	GetGroupKeyRun() GetGroupKeyRunRepository
	Github() GithubRepository
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
