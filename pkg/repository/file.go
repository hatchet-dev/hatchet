package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type CreateFileOpts struct {
	// (required) the tenant id
	TenantId string `validate:"required,uuid"`

	// (required) the file name
	FileName string `validate:"required"`

	// (required) the path to the file
	FilePath string `validate:"required"`

	// (optional) the event metadata
	AdditionalMetadata []byte
}

type FileAPIRepository interface {
	// CreateFile creates a new file for a given tenant.
	CreateFile(ctx context.Context, opts *CreateFileOpts) (*dbsqlc.File, error)

	// GetFileByID returns an file by id.
	GetFileByID(id string) (*dbsqlc.File, error)

	// ListFiles returns all files for a given tenant.
	ListFiles(ctx context.Context, tenantId string, ids []string) ([]*dbsqlc.File, error)
}
