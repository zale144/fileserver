package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

type File struct {
	FileName    string   `json:"fileName"`
	FileContent string   `json:"fileContent"`
	MerkleProof []string `json:"merkleProof"`
}

type MerkleProof struct {
	Index int64    `json:"index"`
	Proof []string `json:"proof"`
}

func DownloadFile(fileID, url string) error {
	response, err := http.Get(fmt.Sprintf("%s/%s", url, fileID))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %v", response.Status)
	}

	file := new(File)
	if err := json.NewDecoder(response.Body).Decode(file); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Create the file
	out, err := os.Create(file.FileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	decoded, err := base64.StdEncoding.DecodeString(file.FileContent)
	if err != nil {
		return fmt.Errorf("failed to decode file: %w", err)
	}

	// Write the body to file
	_, err = io.Copy(out, bytes.NewBuffer(decoded))
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	proof := new(MerkleProof)
	proof.Proof = make([]string, len(file.MerkleProof))
	for i, p := range file.MerkleProof {
		decodedProof, err := base64.StdEncoding.DecodeString(p)
		if err != nil {
			return fmt.Errorf("failed to decode proof: %w", err)
		}
		proof.Proof[i] = fmt.Sprintf("%x", decodedProof)
	}

	proof.Index, err = strconv.ParseInt(file.FileName, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse index: %w", err)
	}

	// Write the proof to file
	proofJsn, err := json.Marshal(proof)
	if err != nil {
		return fmt.Errorf("failed to marshal proof: %w", err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s.proof", file.FileName), proofJsn, 0644); err != nil {
		return fmt.Errorf("failed to write proof: %w", err)
	}

	return nil
}
