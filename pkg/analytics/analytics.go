package analytics

type Analytics interface {
	Enqueue(event string, userId string, tenantId *string, data map[string]interface{})
	Tenant(tenantId string, data map[string]interface{})
}

type NoOpAnalytics struct{}

func (a NoOpAnalytics) Enqueue(event string, userId string, tenantId *string, data map[string]interface{}) {
}
func (a NoOpAnalytics) Tenant(tenantId string, data map[string]interface{}) {}
