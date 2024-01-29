package client

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func UploadFile(filePath, url string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}
	writer.Close()

	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %v", response.Status)
	}

	return nil
}

func UploadDirectory(directoryPath, url string) error {
	// Setup a pipe - this will allow us to pass the multipart writer directly into the request
	pr, pw := io.Pipe()
	w := multipart.NewWriter(pw)
	done := make(chan error)

	go func() {
		defer func() {
			pw.Close()
			w.Close()
			close(done)
		}()

		// Walk through the directory and upload all files
		err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("cannot open file %v: %w", path, err)
				}
				defer file.Close()

				fw, err := w.CreateFormFile("files", filepath.Base(path))
				if err != nil {
					return fmt.Errorf("cannot create form file: %w", err)
				}

				if _, err = io.Copy(fw, file); err != nil {
					return fmt.Errorf("cannot write file to form: %w", err)
				}
				fmt.Printf("file %v uploaded\n", path)
			}
			return nil
		})

		if err != nil {
			done <- fmt.Errorf("error walking through files: %w", err)
			return
		}
	}()

	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{}
	fmt.Println("Uploading directory...")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	if err := <-done; err != nil {
		return fmt.Errorf("error uploading directory: %w", err)
	}
	return nil
}
