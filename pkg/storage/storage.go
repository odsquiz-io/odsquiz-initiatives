// pkg/storage/storage.go: stores objects in a Google Cloud Storage bucket.
package storage

import (
	"context"
	"errors"
	"io"

	gcs "cloud.google.com/go/storage"
)

var (
	ErrUnavailable = errors.New("storage is not configured")
	ErrNotFound    = errors.New("object not found")
)

type Storage interface {
	Put(ctx context.Context, name, contentType string, data []byte) error
	Get(ctx context.Context, name string) ([]byte, error)
	Delete(ctx context.Context, name string) error
}

// Bucket is a Storage backed by a Cloud Storage bucket. Without a bucket every
// call fails with ErrUnavailable, so the service can run without storage.
type Bucket struct {
	bucket *gcs.BucketHandle
}

// New uses Application Default Credentials, or the emulator named by
// STORAGE_EMULATOR_HOST. An empty name returns a Bucket that is unavailable;
// so does a failure to create the client, together with the error.
func New(ctx context.Context, name string) (*Bucket, error) {
	if name == "" {
		return &Bucket{}, nil
	}

	client, err := gcs.NewClient(ctx)
	if err != nil {
		return &Bucket{}, err
	}

	return &Bucket{bucket: client.Bucket(name)}, nil
}

func (b *Bucket) Put(ctx context.Context, name, contentType string, data []byte) error {
	if b.bucket == nil {
		return ErrUnavailable
	}

	writer := b.bucket.Object(name).NewWriter(ctx)
	writer.ContentType = contentType
	writer.ChunkSize = 0

	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return err
	}

	return writer.Close()
}

func (b *Bucket) Get(ctx context.Context, name string) ([]byte, error) {
	if b.bucket == nil {
		return nil, ErrUnavailable
	}

	reader, err := b.bucket.Object(name).NewReader(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	defer func() { _ = reader.Close() }()

	return io.ReadAll(reader)
}

func (b *Bucket) Delete(ctx context.Context, name string) error {
	if b.bucket == nil {
		return ErrUnavailable
	}

	return mapError(b.bucket.Object(name).Delete(ctx))
}

func mapError(err error) error {
	if errors.Is(err, gcs.ErrObjectNotExist) {
		return ErrNotFound
	}

	return err
}
