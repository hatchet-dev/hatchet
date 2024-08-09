package files

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *FileService) FileCreate(ctx echo.Context, request gen.FileCreateRequestObject) (gen.FileCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	var bucketName = "testing-bucket"
	var accountId = "<accountid>"
	var accessKeyId = "<access_key_id>"
	var accessKeySecret = "<access_key_secret>"

	// marshal the data object to bytes
	dataBytes, err := json.Marshal(request.Body.Data)
	if err != nil {
		return nil, err
	}
	// load default s3 config
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-west-2"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
	)

	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	// override default endpoint to use Cloudflare's S3-compatible API
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId))
	})

	key := *request.Body.Filename

	putObjectOutput, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(dataBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to upload file to S3, %v", err)
	}

	// Extract metadata from the S3 response
	additionalMetadata, err := json.Marshal(putObjectOutput.ResultMetadata)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal metadata, %v", err)
	}

	createOpts := &repository.CreateFileOpts{
		TenantId:           tenant.ID,
		FileName:           *request.Body.Filename,
		AdditionalMetadata: additionalMetadata,
	}

	// write the file to the db
	file, err := t.config.APIRepository.File().CreateFile(ctx.Request().Context(), createOpts)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return gen.FileCreate200JSONResponse(
		*transformers.ToFileFromSQLC(file),
	), nil
}
