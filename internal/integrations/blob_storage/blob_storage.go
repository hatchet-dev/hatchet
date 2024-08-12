package blob_storage

import "context"

type BlobStorageService interface {
	Enabled() bool
	PutObject(ctx context.Context, key string, data []byte) ([]byte, error)
	GetObject(ctx context.Context, key string) ([]byte, error)
}

type NoOpService struct{}

func (s *NoOpService) Enabled() bool {
	return false
}

func (s *NoOpService) PutObject(ctx context.Context, key string, data []byte) ([]byte, error) {
	return nil, nil
}

func (s *NoOpService) GetObject(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}
