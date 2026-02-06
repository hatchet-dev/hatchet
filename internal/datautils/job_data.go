package datautils

type TriggeredBy string

const (
	TriggeredByEvent    TriggeredBy = "event"
	TriggeredByCron     TriggeredBy = "cron"
	TriggeredBySchedule TriggeredBy = "schedule"
	TriggeredByManual   TriggeredBy = "manual"
	TriggeredByParent   TriggeredBy = "parent"
)

type JobRunLookupData struct {
	Input       map[string]interface{}            `json:"input"`
	Steps       map[string]map[string]interface{} `json:"steps,omitempty"`
	TriggeredBy TriggeredBy                       `json:"triggered_by"`
}

type StepRunData struct {
	Input         map[string]interface{}            `json:"input"`
	Parents       map[string]map[string]interface{} `json:"parents"`
	UserData      map[string]interface{}            `json:"user_data"`
	Overrides     map[string]interface{}            `json:"overrides"`
	StepRunErrors map[string]string                 `json:"step_run_errors,omitempty"`
	TriggeredBy   TriggeredBy                       `json:"triggered_by"`
}
