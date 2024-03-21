package repository

type APIRepository interface {
	Health() HealthRepository
	APIToken() APITokenRepository
	Event() EventAPIRepository
	Log() LogsAPIRepository
	Tenant() TenantAPIRepository
	TenantInvite() TenantInviteRepository
	Workflow() WorkflowAPIRepository
	WorkflowRun() WorkflowRunAPIRepository
	JobRun() JobRunAPIRepository
	StepRun() StepRunAPIRepository
	Github() GithubRepository
	SNS() SNSRepository
	Step() StepRepository
	Worker() WorkerAPIRepository
	UserSession() UserSessionRepository
	User() UserRepository
}

type EngineRepository interface {
	Health() HealthRepository
	APIToken() EngineTokenRepository
	Dispatcher() DispatcherEngineRepository
	Event() EventEngineRepository
	GetGroupKeyRun() GetGroupKeyRunEngineRepository
	JobRun() JobRunEngineRepository
	StepRun() StepRunEngineRepository
	Tenant() TenantEngineRepository
	Ticker() TickerEngineRepository
	Worker() WorkerEngineRepository
	Workflow() WorkflowEngineRepository
	WorkflowRun() WorkflowRunEngineRepository
	Log() LogsEngineRepository
}

func BoolPtr(b bool) *bool {
	return &b
}

func StringPtr(s string) *string {
	return &s
}
