package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type CreateFileOpts struct {
	// (required) the tenant id
	TenantId string `validate:"required,uuid"`

	// (required) the event key
	Key string `validate:"required"`

	// (required) the file name
	FileName string `validate:"required"`

	// (required) the path to the file
	FilePath string `validate:"required"`

	// (optional) the event metadata
	AdditionalMetadata []byte
}

type ListFileOpts struct {
	// (optional) number of files to skip
	Offset *int

	// (optional) number of files to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`
}

type FileAPIRepository interface {
	// CreateFile creates a new file for a given tenant.
	CreateFile(ctx context.Context, opts *CreateFileOpts) (*dbsqlc.File, error)

	// ListFiles returns all files for a given tenant.
	ListFiles(tenantId string, opts *ListFileOpts) ([]*dbsqlc.File, error)
}
