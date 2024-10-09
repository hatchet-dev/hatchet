package alerttypes

type ResourceLimitAlert struct {
	Link          string `json:"link"`
	Resource      string `json:"resource"`
	AlertType     string `json:"alert_type"`
	CurrentValue  int    `json:"current_value"`
	LimitValue    int    `json:"limit_value"`
	Percentage    int    `json:"percentage"`
	LastRefillAgo string `json:"refill_date_time"`
	LimitWindow   string `json:"limit_window"`
}
