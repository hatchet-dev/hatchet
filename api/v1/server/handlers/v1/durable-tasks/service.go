package durabletasks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type DurableTasksService struct {
	config *server.ServerConfig
}

func NewDurableTasksService(config *server.ServerConfig) *DurableTasksService {
	return &DurableTasksService{
		config: config,
	}
}

type waitForConditionData struct {
	Type       string `json:"type"`
	OrGroupID  string `json:"orGroupId"`
	DataKey    string `json:"dataKey"`
	SleepFor   string `json:"sleepFor,omitempty"`
	EventKey   string `json:"eventKey,omitempty"`
	Expression string `json:"expression,omitempty"`
}

func unmarshalConditions(data []byte, out *[]waitForConditionData) error {
	return json.Unmarshal(data, out)
}

func buildWaitForDescription(conditions []waitForConditionData) string {
	if len(conditions) == 0 {
		return "Waiting"
	}

	groupOrder := make([]string, 0)
	groups := make(map[string][]waitForConditionData)
	for _, c := range conditions {
		if _, seen := groups[c.OrGroupID]; !seen {
			groupOrder = append(groupOrder, c.OrGroupID)
		}
		groups[c.OrGroupID] = append(groups[c.OrGroupID], c)
	}

	groupDescs := make([]string, 0, len(groupOrder))
	for _, gid := range groupOrder {
		conds := groups[gid]
		parts := make([]string, 0, len(conds))
		for _, c := range conds {
			parts = append(parts, describeCondition(c))
		}
		groupDescs = append(groupDescs, strings.Join(parts, " or "))
	}

	return "Waiting for " + strings.Join(groupDescs, " and ")
}

func describeCondition(c waitForConditionData) string {
	switch c.Type {
	case "SLEEP":
		if c.SleepFor != "" {
			return fmt.Sprintf("sleep(%s)", c.SleepFor)
		}
		return "sleep"
	case "USER_EVENT":
		if c.EventKey != "" {
			return fmt.Sprintf("event %q", c.EventKey)
		}
		return "user event"
	}
	return c.Type
}
