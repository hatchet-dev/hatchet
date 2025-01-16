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
	TriggeredBy TriggeredBy                       `json:"triggered_by"`
	Steps       map[string]map[string]interface{} `json:"steps,omitempty"`
}

type StepRunData struct {
	Input       map[string]interface{}            `json:"input"`
	TriggeredBy TriggeredBy                       `json:"triggered_by"`
	Parents     map[string]map[string]interface{} `json:"parents"`

	// custom-set user data for the step
	UserData map[string]interface{} `json:"user_data"`

	// overrides set from the playground
	Overrides map[string]interface{} `json:"overrides"`

	// errors in upstream steps (only used in on-failure step)
	StepRunErrors map[string]string `json:"step_run_errors,omitempty"`
}
