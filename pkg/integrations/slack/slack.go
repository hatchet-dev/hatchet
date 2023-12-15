package slack

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/pkg/integrations"
	"github.com/slack-go/slack"
)

type SlackIntegration struct {
	api              *slack.Client
	teamId           string
	decoderValidator datautils.DataDecoderValidator
}

func NewSlackIntegration(authToken string, teamId string, debug bool) *SlackIntegration {
	api := slack.New(authToken, slack.OptionDebug(debug))
	decoderValidator := datautils.NewDataDecoderValidator()

	return &SlackIntegration{
		api:              api,
		teamId:           teamId,
		decoderValidator: decoderValidator,
	}
}

func (s *SlackIntegration) GetId() string {
	return "slack"
}

func (s *SlackIntegration) Actions() []string {
	return []string{
		"create-channel",
		"send-message",
		"add-users-to-channel",
	}
}

func (s *SlackIntegration) ActionHandler(action string) any {
	switch action {
	case "create-channel":
		return s.createChannel
	case "add-users-to-channel":
		return s.addUsersToChannel
	case "send-message":
		return s.sendMessageToChannel
	default:
		return nil
	}
}

func (s *SlackIntegration) GetWebhooks() []integrations.IntegrationWebhook {
	return []integrations.IntegrationWebhook{}
}

type CreateChannelData struct {
	ChannelName string `json:"channelName"`
}

func (s *SlackIntegration) createChannel(ctx context.Context, data *CreateChannelData) (map[string]interface{}, error) {
	channel, err := s.api.CreateConversation(slack.CreateConversationParams{
		IsPrivate:   true,
		ChannelName: data.ChannelName,
		TeamID:      s.teamId,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating slack channel: %w", err)
	}

	return map[string]interface{}{
		"channelId": channel.ID,
	}, nil
}

type AddUsersToChannelData struct {
	ChannelID string   `json:"channelId"`
	UserIDs   []string `json:"userIds"`
}

func (s *SlackIntegration) addUsersToChannel(ctx context.Context, data *AddUsersToChannelData) (map[string]interface{}, error) {
	_, err := s.api.InviteUsersToConversation(data.ChannelID, data.UserIDs...)

	if err != nil {
		return nil, fmt.Errorf("error adding users to slack channel: %w", err)
	}

	return map[string]interface{}{}, nil
}

type SendMessageToChannelData struct {
	ChannelID string `json:"channelId"`
	Message   string `json:"message"`
}

func (s *SlackIntegration) sendMessageToChannel(ctx context.Context, data *SendMessageToChannelData) (map[string]interface{}, error) {
	_, _, err := s.api.PostMessage(data.ChannelID, slack.MsgOptionText(data.Message, false))

	if err != nil {
		return nil, fmt.Errorf("error sending message to slack channel: %w", err)
	}

	return map[string]interface{}{}, nil
}
