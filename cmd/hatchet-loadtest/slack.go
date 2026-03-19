package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type SlackSender struct {
	s3Client        *s3.Client
	s3Bucket        string
	slackWebhookUrl string
}

func NewS3Client(ctx context.Context) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-west-2"),
	)
	if err != nil {
		panic(err)
	}

	client := s3.NewFromConfig(cfg)
	return client, nil
}

func NewSlackSender(s3Bucket string, slackWebhookUrl string) *SlackSender {
	s3Client, _ := NewS3Client(context.Background())
	return &SlackSender{
		s3Client:        s3Client,
		s3Bucket:        s3Bucket,
		slackWebhookUrl: slackWebhookUrl,
	}
}

func (s *SlackSender) SendMessage(durationPlotUrl string, schedulingPlotUrl string, avgDuration time.Duration, avgScheduling time.Duration) error {
	// Create the payload
	payload := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*(%s)* \n:star:New load test results:star:\nAverage task duration: %s\nAverage task scheduling: %s", time.Now().Format("2006-01-02-15:04:05"), avgDuration.String(), avgScheduling.String()),
				},
			},
			{
				"type":      "image",
				"image_url": schedulingPlotUrl,
				"alt_text":  "Scheduling test graph",
			},
			{
				"type":      "image",
				"image_url": durationPlotUrl,
				"alt_text":  "Duration test graph",
			},
		},
	}
	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	resp, err := http.Post(s.slackWebhookUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("received non-2xx status code: %d", resp.StatusCode)
	}
	return nil
}

func (s *SlackSender) UploadS3(imageBytes []byte) (*string, error) {
	key := fmt.Sprintf("%s-%s", "loadtest-plot", time.Now().Format("20060102150405"))
	_, err := s.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.s3Bucket),
		Key:    &key,
		Body:   bytes.NewReader(imageBytes),
	})
	if err != nil {
		return nil, err
	}
	presigner := s3.NewPresignClient(s.s3Client)
	req, err := presigner.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &s.s3Bucket,
		Key:    &key,
	})
	if err != nil {
		panic(err)
	}

	uploadedUrl := req.URL
	return &uploadedUrl, nil
}

func (s *SlackSender) SendToSlack(durationBytes []byte, schedulingBytes []byte, avgDuration time.Duration, avgScheduling time.Duration) error {
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
