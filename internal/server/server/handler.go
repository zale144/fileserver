package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/zale144/fileserver/internal/server/model"
	"go.uber.org/zap"
)

type fileService interface {
	Get(ctx context.Context, index int) (*model.File, error)
	SaveStream(ctx context.Context, fileCh chan *model.IndexedFileInput) error
	Verify(fileMD *model.File, fileHash, merkleRoot []byte) error
}
type FileUploadResponse struct {
	Status string `json:"status"`
}

type FileDownloadResponse struct {
	FileName    string   `json:"fileName"`
	FileContent []byte   `json:"fileContent"`
	MerkleProof [][]byte `json:"merkleProof"`
}

func (s *Server) DownloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["index"], 10, 64)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	file, err := s.fileSvc.Get(r.Context(), int(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "File not Found", http.StatusNotFound)
			return
		}
		s.log.Error("error getting file", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := FileDownloadResponse{
		FileName:    fmt.Sprintf("%d", id),
		FileContent: file.Data,
		MerkleProof: file.Metadata.MerkleProof,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error Writing Response", http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(file.Data); err != nil {
		s.log.Error("error writing file to response", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) UploadMultiple(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	reader, err := r.MultipartReader()
	if err != nil {
		s.log.Error("error getting multipart reader", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	fileCh := make(chan *model.IndexedFileInput)
	go func() {
		defer close(fileCh)
		i := 0
		for {
			part, err := reader.NextPart()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				s.log.Error("error getting next part", zap.Error(err))
				return
			}
			if part.FileName() == "" {
				continue
			}
			b := bytes.NewBuffer(nil)
			_, _ = b.ReadFrom(part)

			index := i
			idx, err := strconv.ParseInt(part.FileName(), 10, 64)
			if err == nil {
				index = int(idx)
			}
			fileCh <- &model.IndexedFileInput{
				Index: index,
				Data:  b.Bytes(),
			}
			_ = part.Close()
			i++
		}

	}()

	if err := s.fileSvc.SaveStream(r.Context(), fileCh); err != nil {
		s.log.Error("error saving file", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := FileUploadResponse{
		Status: "Success",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("error encoding response", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// Interface assertions.
var (
	_ http.HandlerFunc = (*Server)(nil).DownloadFile
	_ http.HandlerFunc = (*Server)(nil).UploadMultiple
)
