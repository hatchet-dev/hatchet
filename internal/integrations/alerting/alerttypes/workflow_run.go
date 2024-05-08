package alerttypes

type WorkflowRunFailedItem struct {
	Link                  string `json:"link"`
	WorkflowName          string `json:"workflow_name"`
	WorkflowRunReadableId string `json:"workflow_run_readable_id"`
	RelativeDate          string `json:"relative_date"`
	AbsoluteDate          string `json:"absolute_date"`
}
