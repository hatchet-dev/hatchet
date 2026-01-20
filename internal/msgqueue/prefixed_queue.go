package msgqueue

type prefixedQueue struct {
	prefix string
	inner  Queue
}

func PrefixQueue(prefix string, q Queue) Queue {
	return prefixedQueue{
		prefix: prefix,
		inner:  q,
	}
}

func (p prefixedQueue) Name() string {
	return p.prefix + p.inner.Name()
}

func (p prefixedQueue) Durable() bool {
	return p.inner.Durable()
}

func (p prefixedQueue) AutoDeleted() bool {
	return p.inner.AutoDeleted()
}

func (p prefixedQueue) Exclusive() bool {
	return p.inner.Exclusive()
}

func (p prefixedQueue) FanoutExchangeKey() string {
	return p.inner.FanoutExchangeKey()
}

func (p prefixedQueue) DLQ() Queue {
	dlq := p.inner.DLQ()
	if dlq == nil {
		return nil
	}

	return prefixedQueue{
		prefix: p.prefix,
		inner:  dlq,
	}
}

func (p prefixedQueue) IsDLQ() bool {
	return p.inner.IsDLQ()
}

func (p prefixedQueue) IsAutoDLQ() bool {
	return p.inner.IsAutoDLQ()
}

func (p prefixedQueue) IsExpirable() bool {
	return p.inner.IsExpirable()
}

