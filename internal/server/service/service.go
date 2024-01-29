package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/zale144/fileserver/internal/merkle"
	"go.uber.org/zap"

	"github.com/zale144/fileserver/internal/server/model"
)

type File struct {
	repo    fileRepository
	storage fileStorage
	log     *zap.Logger
}

type fileRepository interface {
	Get(index int) (*model.FileMetadata, error)
	PutMultiple(ctx context.Context, md <-chan *model.FileMetadata) error
}

type fileStorage interface {
	Download(ctx context.Context, path string) ([]byte, error)
	UploadMultiple(ctx context.Context, dataCh <-chan *model.File) error
}

func NewFile(repo fileRepository, storage fileStorage, log *zap.Logger) *File {
	return &File{
		repo:    repo,
		storage: storage,
		log:     log,
	}
}

func (f *File) Get(ctx context.Context, index int) (*model.File, error) {
	fileMD, err := f.repo.Get(index)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from repo: %w", err)
	}

	hash := fmt.Sprintf("%x", fileMD.Hash)
	data, err := f.storage.Download(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from storage: %w", err)
	}

	file := &model.File{
		Data:     data,
		Metadata: fileMD,
	}

	return file, nil
}

func (f *File) SaveStream(ctx context.Context, inCh chan *model.IndexedFileInput) error {
	fileCh := make(chan *model.File, 1)
	fileMDCh := make(chan *model.FileMetadata, 1)
	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}

	data, proofs := getMerkleProofs(inCh)

	go func(data [][]byte, proofs [][][]byte) {
		defer func() {
			close(fileCh)
			close(fileMDCh)
		}()
		toFile(ctx, data, proofs, fileCh, fileMDCh)
	}(data, proofs)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := f.storage.UploadMultiple(ctx, fileCh); err != nil {
			f.log.Error("failed to save file", zap.Error(err))
			errCh <- err
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := f.repo.PutMultiple(ctx, fileMDCh); err != nil {
			f.log.Error("failed to save file metadata", zap.Error(err))
			errCh <- err
			return
		}
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			return fmt.Errorf("failed to save file: %w", err)
		}
	}
	return nil
}

func toFile(ctx context.Context, data [][]byte, proofs [][][]byte, fileCh chan *model.File, fileMDCh chan *model.FileMetadata) {
	for i, d := range data {
		hash := merkle.HashData(d)
		file := &model.File{
			Data: d,
			Metadata: &model.FileMetadata{
				Index:       i,
				Hash:        hash,
				MerkleProof: proofs[i],
			},
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		fileCh <- file
		fileMDCh <- file.Metadata
	}
}

func getMerkleProofs(inCh chan *model.IndexedFileInput) ([][]byte, [][][]byte) {
	var data [][]byte
	for file := range inCh {
		data = append(data, file.Data)
	}
	tree := merkle.NewTree(data)
	return data, tree.Proofs
}

func (f *File) Verify(fileMD *model.File, fileHash, root []byte) error {
	index := fileMD.Metadata.Index
	proof := fileMD.Metadata.MerkleProof
	valid := merkle.VerifyProof(index, fileHash, proof, root)
	if !valid {
		return fmt.Errorf("file verification failed")
	}
	return nil
}
