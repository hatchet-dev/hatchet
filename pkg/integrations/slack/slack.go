package slack

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"

	"github.com/hatchet-dev/hatchet/pkg/integrations"
)

type SlackIntegration struct {
	api    *slack.Client
	teamId string
}

func NewSlackIntegration(authToken string, teamId string, debug bool) *SlackIntegration {
	api := slack.New(authToken, slack.OptionDebug(debug))

	return &SlackIntegration{
		api:    api,
		teamId: teamId,
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

type CreateChannelOutput struct {
	ChannelId string `json:"channelId"`
}

func (s *SlackIntegration) createChannel(ctx context.Context, data *CreateChannelData) (*CreateChannelOutput, error) {
	channel, err := s.api.CreateConversation(slack.CreateConversationParams{
		IsPrivate:   true,
		ChannelName: data.ChannelName,
		TeamID:      s.teamId,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating slack channel: %w", err)
	}

	return &CreateChannelOutput{
		ChannelId: channel.ID,
	}, nil
}

type AddUsersToChannelData struct {
	ChannelID string   `json:"channelId"`
	UserIDs   []string `json:"userIds"`
}

type AddUsersToChannelOutput struct{}

func (s *SlackIntegration) addUsersToChannel(ctx context.Context, data *AddUsersToChannelData) error {
	_, err := s.api.InviteUsersToConversation(data.ChannelID, data.UserIDs...)

	if err != nil {
		return fmt.Errorf("error adding users to slack channel: %w", err)
	}

	return nil
}

type SendMessageToChannelData struct {
	ChannelID string `json:"channelId"`
	Message   string `json:"message"`
}

func (s *SlackIntegration) sendMessageToChannel(ctx context.Context, data *SendMessageToChannelData) error {
	_, _, err := s.api.PostMessage(data.ChannelID, slack.MsgOptionText(data.Message, false))

	if err != nil {
		return fmt.Errorf("error sending message to slack channel: %w", err)
	}

	return nil
}
