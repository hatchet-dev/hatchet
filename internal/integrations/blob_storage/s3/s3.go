package blob_storage_s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type S3Client struct {
	BucketName      string
	Region          string
	AccessKeyId     string
	AccessKeySecret string
	BaseEndpoint    *string
}

func NewS3Client(config server.S3ConfigFile) (*S3Client, error) {
	return &S3Client{
		BucketName:      config.BucketName,
		Region:          config.Region,
		AccessKeyId:     config.AccessKeyId,
		AccessKeySecret: config.AccessKeySecret,
		BaseEndpoint:    config.BaseEndpoint,
	}, nil
}

func (s *S3Client) Enabled() bool {
	return true
}

func (s *S3Client) getClient() (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(s.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s.AccessKeyId, s.AccessKeySecret, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if s.BaseEndpoint != nil {
			o.BaseEndpoint = aws.String(*s.BaseEndpoint)
		}
	})

	return client, nil
}

func (s *S3Client) PutObject(ctx context.Context, key string, data []byte) ([]byte, error) {

	// override default endpoint to use Cloudflare's S3-compatible API
	client, err := s.getClient()

	if err != nil {
		return nil, fmt.Errorf("unable to get S3 client, %v", err)
	}

	putObjectOutput, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})

	if err != nil {
		return nil, fmt.Errorf("unable to upload file to S3, %v", err)
	}

	// Extract metadata from the S3 response
	additionalMetadata, err := json.Marshal(putObjectOutput.ResultMetadata)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal metadata, %v", err)
	}

	return additionalMetadata, nil
}

func (s *S3Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	client, err := s.getClient()

	if err != nil {
		return nil, fmt.Errorf("unable to get S3 client, %v", err)
	}

	getObjectOutput, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get file from S3, %v", err)
	}
	defer getObjectOutput.Body.Close()

	// Read the file content
	fileContent, err := io.ReadAll(getObjectOutput.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read file content, %v", err)
	}

	return fileContent, nil
}
