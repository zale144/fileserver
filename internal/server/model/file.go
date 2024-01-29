package model

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type File struct {
	Data     []byte
	Metadata *FileMetadata
}

type FileMetadata struct {
	Index       int        `db:"index"`
	Hash        []byte     `db:"hash"`
	MerkleProof ByteaArray `db:"merkle_proof"`
}

type IndexedFileInput struct {
	Index int
	Data  []byte
}

type ByteaArray [][]byte

func (p *ByteaArray) Scan(src any) error {
	if src == nil {
		*p = nil
		return nil
	}

	switch src := src.(type) {
	case []byte:
		parsed, err := parseByteaArray(string(src))
		if err != nil {
			return fmt.Errorf("failed to parse bytea array: %w", err)
		}
		*p = parsed
	default:
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type %T", src, *p)
	}

	return nil
}

func parseByteaArray(byteaStr string) ([][]byte, error) {
	byteaStr = strings.Trim(byteaStr, "{}")
	hexStrs := strings.Split(byteaStr, ",")
	result := make([][]byte, len(hexStrs))
	for i, hexStr := range hexStrs {
		hexStr = strings.Trim(hexStr, "\"")
		hexStr = strings.TrimPrefix(hexStr, `\\x`)
		hexStr = strings.TrimPrefix(hexStr, "27")
		hexStr = strings.TrimSuffix(hexStr, "27")
		bytes, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, err
		}
		decodedBytes := make([]byte, hex.DecodedLen(len(bytes)))
		_, err = hex.Decode(decodedBytes, bytes)
		if err != nil {
			return nil, err
		}
		result[i] = decodedBytes
	}
	return result, nil
}
