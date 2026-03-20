package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/slack-go/slack"
)

type SlackSender struct {
	s3Client *s3.Client
	s3Bucket string
	Token    string
	Channel  string
	Thread   string
}

func NewS3Client(ctx context.Context) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-west-2"),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	return client, nil
}

func ShouldSendSlack() bool {
	return (os.Getenv("SLACK_BOT_TOKEN") != "" &&
		os.Getenv("SLACK_THREAD_TS") != "" &&
		os.Getenv("SLACK_CHANNEL_ID") != "" &&
		os.Getenv("AWS_ACCESS_KEY_ID") != "" &&
		os.Getenv("AWS_SECRET_ACCESS_KEY") != "")
}

func NewSlackSender(s3Bucket string) *SlackSender {
	s3Client, _ := NewS3Client(context.Background())
	return &SlackSender{
		s3Client: s3Client,
		s3Bucket: s3Bucket,
		Token:    os.Getenv("SLACK_BOT_TOKEN"),
		Thread:   os.Getenv("SLACK_THREAD_TS"),
		Channel:  os.Getenv("SLACK_CHANNEL_ID"),
	}
}

func (s *SlackSender) SendMessage(durationPlotUrl string, schedulingPlotUrl string, avgDuration time.Duration, avgScheduling time.Duration) error {
	text := fmt.Sprintf(
		":star:Load test results:star:\nAverage task duration: %s\nAverage task scheduling: %s",
		avgDuration.String(),
		avgScheduling.String(),
	)

	section := slack.NewSectionBlock(
		slack.NewTextBlockObject("mrkdwn", text, false, false),
		nil,
		nil,
	)

	image1 := slack.NewImageBlock(
		schedulingPlotUrl,
		"Scheduling test graph",
		"",
		nil,
	)

	image2 := slack.NewImageBlock(
		durationPlotUrl,
		"Duration test graph",
		"",
		nil,
	)

	blocks := []slack.Block{section, image1, image2}
	client := slack.New(s.Token)
	_, _, err := client.PostMessage(
		s.Channel,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(s.Thread), // 👈 this attaches it to a thread
	)
	return err
}

func (s *SlackSender) UploadS3(imageBytes []byte) (*string, error) {
	key := fmt.Sprintf("%s-%s-%s", "loadtest-plot", uuid.New(), time.Now().Format("20060102150405"))
	_, err := s.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.s3Bucket),
		Key:    &key,
		Body:   bytes.NewReader(imageBytes),
	})
	if err != nil {
		return nil, err
	}
	presigner := s3.NewPresignClient(s.s3Client)
	req, err := presigner.PresignGetObject(context.Background(),
		&s3.GetObjectInput{
			Bucket: &s.s3Bucket,
			Key:    &key,
		},
		s3.WithPresignExpires(time.Hour*24*7),
	)
	if err != nil {
		return nil, err
	}

	uploadedUrl := req.URL
	return &uploadedUrl, nil
}

func (s *SlackSender) Send(durationBytes []byte, schedulingBytes []byte, avgDuration time.Duration, avgScheduling time.Duration) error {
	uploadedDurationFileUrl, err := s.UploadS3(durationBytes)
	if err != nil {
		return err
	}
	uploadedSchedulingFileUrl, err := s.UploadS3(schedulingBytes)
	if err != nil {
		return err
	}
	return s.SendMessage(*uploadedDurationFileUrl, *uploadedSchedulingFileUrl, avgDuration, avgScheduling)
}
