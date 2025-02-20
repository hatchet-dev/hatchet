package alerttypes

type ExpiringTokenItem struct {
	Link                  string `json:"link"`
	TokenName             string `json:"token_name"`
	ExpiresAtRelativeDate string `json:"expires_at_relative_date"`
	ExpiresAtAbsoluteDate string `json:"expires_at_absolute_date"`
}
