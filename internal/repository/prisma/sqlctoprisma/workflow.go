package sqlctoprisma

import (
	"encoding/json"
)

type Converter[K any, V any] struct{}

func NewConverter[K any, V any]() *Converter[K, V] {
	return &Converter[K, V]{}
}

func (c *Converter[K, V]) convert(opt *K) *V {
	bytes, err := json.Marshal(opt)

	if err != nil {
		return nil
	}

	var res V

	err = json.Unmarshal(bytes, &res)

	if err != nil {
		return nil
	}

	return &res
}

func (c *Converter[K, V]) ToPrismaList(list []*K) []*V {
	res := make([]*V, len(list))

	for i, item := range list {
		item := c.convert(item)

		res[i] = item
	}

	return res
}

func (c *Converter[K, V]) ToPrisma(opt *K) *V {
	return c.convert(opt)
}
