package files

import (
	"encoding/json"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *FileService) FileCreate(ctx echo.Context, request gen.FileCreateRequestObject) (gen.FileCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	if !t.config.BlobStorage.Enabled() {
		return nil, fmt.Errorf("blob storage is not enabled")
	}

	// marshal the data object to bytes
	dataBytes, err := json.Marshal(request.Body.Data)
	if err != nil {
		return nil, err
	}

	context := ctx.Request().Context()
	key := *request.Body.Filename
	additionalMetadata, err := t.config.BlobStorage.PutObject(context, key, dataBytes)

	if err != nil {
		return nil, fmt.Errorf("unable to upload file: %v", err)
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
