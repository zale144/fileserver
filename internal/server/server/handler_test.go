package server

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/zale144/fileserver/internal/server/model"
	"github.com/zale144/fileserver/internal/server/service"
	"go.uber.org/zap"
)

func TestUploadMultiple(t *testing.T) {
	log := zap.NewNop()
	tests := []struct {
		name           string
		numFiles       int
		uploadService  *mockStorageService
		repositorySvc  *mockRepositoryService
		wantError      error
		wantStatusCode int
	}{
		{
			name:           "Successful Save 1 file",
			numFiles:       1,
			wantStatusCode: http.StatusOK,
		}, {
			name:           "Successful Save 3 files",
			numFiles:       3,
			wantStatusCode: http.StatusOK,
		}, {
			name:           "Successful Save 100 files",
			numFiles:       100,
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			repositorySvc := newMockRepositoryService()
			uploadService := newMockStorageService(false)
			fileSvc := service.NewFile(repositorySvc, uploadService, log)

			request := createFileUploadRequest(t, tt.numFiles)
			server := Server{fileSvc: fileSvc}
			server.UploadMultiple(rr, request)

			if status := rr.Result().StatusCode; status != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatusCode)
				return
			}

			for i := 0; i < tt.numFiles; i++ {
				file, err := fileSvc.Get(context.Background(), i)
				require.NoError(t, err)
				require.NotNil(t, file)
			}
		})
	}
}

func TestDownloadFile(t *testing.T) {
	log := zap.NewNop()
	tests := []struct {
		name           string
		index          int
		corruptFile    bool
		uploadService  *mockStorageService
		repositorySvc  *mockRepositoryService
		wantError      error
		wantStatusCode int
	}{
		{
			name:           "Successful Download file",
			index:          0,
			wantStatusCode: http.StatusOK,
		}, {
			name:           "File not found",
			index:          99,
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositorySvc := newMockRepositoryService()
			uploadService := newMockStorageService(tt.corruptFile)
			fileSvc := service.NewFile(repositorySvc, uploadService, log)
			inCh := make(chan *model.IndexedFileInput, tt.index)
			go func() {
				defer close(inCh)
				for i := 0; i <= tt.index; i++ {
					inCh <- &model.IndexedFileInput{
						Index: i,
						Data:  bytes.NewBuffer([]byte(fmt.Sprintf("test%d", i))),
					}
				}

			}()

			_, err := fileSvc.SaveStream(context.Background(), inCh)
			require.NoError(t, err)

			req, err := http.NewRequest("GET", fmt.Sprintf("/file/%d", tt.index), nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			server := Server{fileSvc: fileSvc, log: log}
			router := mux.NewRouter()
			router.HandleFunc("/file/{index}", server.DownloadFile)
			router.ServeHTTP(rr, req)

			if status := rr.Result().StatusCode; status != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatusCode)
			}
		})
	}
}

func createFileUploadRequest(t *testing.T, numFiles int) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	defer writer.Close()
	for i := 0; i < numFiles; i++ {
		file, err := writer.CreateFormFile("files", fmt.Sprintf("test%d.txt", i))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = file.Write([]byte(fmt.Sprintf("test%d", i)))
	}
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

type mockStorageService struct {
	m           sync.Map
	corruptFile bool
}

func newMockStorageService(corruptFile bool) *mockStorageService {
	return &mockStorageService{
		corruptFile: corruptFile,
	}
}

func (m *mockStorageService) UploadMultiple(_ context.Context, dataCh <-chan *model.File) error {
	for data := range dataCh {
		if m.corruptFile {
			data.Data = bytes.ReplaceAll(data.Data, []byte("test"), []byte("corrupt"))
		}
		m.m.Store(fmt.Sprintf("%x", data.Metadata.Hash), data)
	}
	return nil
}

func (m *mockStorageService) Download(_ context.Context, id string) ([]byte, error) {
	value, ok := m.m.Load(id)
	if !ok {
		return nil, fmt.Errorf("failed to get file from storage")
	}
	return value.(*model.File).Data, nil
}

type mockRepositoryService struct {
	m sync.Map
}

func newMockRepositoryService() *mockRepositoryService {
	return &mockRepositoryService{}
}

func (m *mockRepositoryService) PutMultiple(_ context.Context, md <-chan *model.FileMetadata) error {
	for data := range md {
		m.m.Store(data.Index, data)
	}
	return nil
}

func (m *mockRepositoryService) Get(index int) (*model.FileMetadata, error) {
	value, ok := m.m.Load(index)
	if !ok {
		return nil, fmt.Errorf("failed to get file from repository")
	}
	return value.(*model.FileMetadata), nil
}
