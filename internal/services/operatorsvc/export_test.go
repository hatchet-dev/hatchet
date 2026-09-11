package operatorsvc

// RetainedResponses reports how many responses the channel's pump holds for Recv, deliverable
// or held, so tests can check the retention bound and the duplicate rule.
func RetainedResponses(ch DurableChannel) int {
	c := ch.(*durableChannel)

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.retained
}
