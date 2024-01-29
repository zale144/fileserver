package storage

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zale144/fileserver/internal/server/model"
)

type File struct {
	minio      *minio.Client
	bucketName string
}

type Config struct {
	BucketName      string `envconfig:"BUCKET_NAME" default:"fileserver"`
	Endpoint        string `envconfig:"MINIO_ENDPOINT" default:"localhost:9000"`
	AccessKeyID     string `envconfig:"MINIO_ACCESS_KEY" default:"minio"`
	SecretAccessKey string `envconfig:"MINIO_SECRET_KEY" default:"minio123"`
}

func NewFile(config Config) (*File, error) {
	minioClient, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &File{
		minio:      minioClient,
		bucketName: config.BucketName,
	}, nil
}

func (f *File) Download(ctx context.Context, name string) ([]byte, error) {
	object, err := f.minio.GetObject(ctx, f.bucketName, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(object); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return buf.Bytes(), nil
}

func (f *File) UploadMultiple(ctx context.Context, dataCh <-chan *model.File) error {
	opts := minio.SnowballOptions{
		Opts: minio.PutObjectOptions{
			ContentType:           "application/octet-stream",
			ConcurrentStreamParts: true,
			NumThreads:            10,
		},
	}
	objects := make(chan minio.SnowballObject, 1)

	go func() {
		defer close(objects)
		for data := range dataCh {
			objects <- minio.SnowballObject{
				Key:     fmt.Sprintf("%x", data.Metadata.Hash),
				Size:    int64(len(data.Data)),
				Content: bytes.NewBuffer(data.Data),
			}
		}
	}()

	if err := f.minio.PutObjectsSnowball(ctx, f.bucketName, opts, objects); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

func (f *File) MakeBucket() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := f.minio.MakeBucket(ctx, f.bucketName, minio.MakeBucketOptions{}); err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := f.minio.BucketExists(ctx, f.bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", f.bucketName)
		} else {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	log.Printf("Successfully created %s\n", f.bucketName)
	return nil
}
