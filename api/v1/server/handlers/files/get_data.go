package files

import (
	"context"
	"fmt"
	"io"

	"github.com/labstack/echo/v4"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func (t *FileService) EventDataGet(ctx echo.Context, request gen.FileDataGetRequestObject) (gen.FileDataGetResponseObject, error) {
	file := ctx.Get("file").(*dbsqlc.File)

	var bucketName = "testing-bucket"
	var accountId = "<accountid>"
	var accessKeyId = "<access_key_id>"
	var accessKeySecret = "<access_key_secret>"

	// Initialize AWS S3 client with custom base endpoint
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-west-2"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId))
	})

	key := file.FileName

	// Get the file from S3
	getObjectOutput, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
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

	dataStr := string(fileContent)

	return gen.FileDataGet200JSONResponse(
		gen.FileData{
			Data: dataStr,
		},
	), nil
}
