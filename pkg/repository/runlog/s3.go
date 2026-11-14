package runlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Config struct {
	Bucket string
	Region string
	// EndpointURL overrides the AWS endpoint (an S3 emulator); it also switches to path-style addressing.
	EndpointURL     string
	AccessKeyID     string
	SecretAccessKey string
}

type S3Store struct {
	client *s3.Client
	bucket string
}

func NewS3Store(ctx context.Context, c S3Config) (*S3Store, error) {
	if c.Bucket == "" || c.Region == "" {
		return nil, fmt.Errorf("run log store needs a bucket and a region")
	}

	loadOpts := []func(*config.LoadOptions) error{config.WithRegion(c.Region)}

	if c.AccessKeyID != "" {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, "")))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)

	if err != nil {
		return nil, fmt.Errorf("could not load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if c.EndpointURL != "" {
			o.BaseEndpoint = aws.String(c.EndpointURL)
			o.UsePathStyle = true
		}

		// ranged GETs of a single bundle cannot be checked against a whole-object checksum
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})

	return &S3Store{client: client, bucket: c.Bucket}, nil
}

func (s *S3Store) Put(ctx context.Context, key string, body []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(body),
		ContentLength: aws.Int64(int64(len(body))),
	})

	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}

	return nil
}

func (s *S3Store) Get(ctx context.Context, key string) ([]byte, error) {
	return s.get(ctx, key, nil)
}

func (s *S3Store) GetRange(ctx context.Context, key string, offset, length int64) ([]byte, error) {
	r := fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)

	body, err := s.get(ctx, key, &r)

	if err != nil {
		return nil, err
	}

	if int64(len(body)) != length {
		return nil, fmt.Errorf("get %s %s: got %d bytes", key, r, len(body))
	}

	return body, nil
}

func (s *S3Store) get(ctx context.Context, key string, byteRange *string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Range:  byteRange,
	})

	if err != nil {
		var noSuchKey *types.NoSuchKey

		if errors.As(err, &noSuchKey) {
			return nil, fmt.Errorf("%s: %w", key, ErrNotFound)
		}

		return nil, fmt.Errorf("get %s: %w", key, err)
	}

	defer out.Body.Close()

	return io.ReadAll(out.Body)
}

func (s *S3Store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err == nil {
		return true, nil
	}

	var notFound *types.NotFound

	if errors.As(err, &notFound) {
		return false, nil
	}

	return false, fmt.Errorf("head %s: %w", key, err)
}
