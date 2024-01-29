package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/zale144/fileserver/internal/server/model"
)

type File struct {
	db *sql.DB
}

func NewFile(db *sql.DB) *File {
	return &File{
		db: db,
	}
}

func (repo *File) Get(index int) (*model.FileMetadata, error) {
	query := `SELECT index, hash, merkle_proof FROM file_metadata WHERE index = $1;`
	row := repo.db.QueryRow(query, index)

	var metadata model.FileMetadata
	err := row.Scan(&metadata.Index, &metadata.Hash, &metadata.MerkleProof)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

const batchSize = 100

func (repo *File) PutMultiple(ctx context.Context, md <-chan *model.FileMetadata) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	values := make([]interface{}, 0, batchSize*3) // 3 fields per record
	valueStrings := make([]string, 0, batchSize)

	count := 0
	for metadata := range md {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", count*3+1, count*3+2, count*3+3))
		merkleProofArray := byteSlicesToByteaArray(metadata.MerkleProof)
		values = append(values, metadata.Index, metadata.Hash, pq.Array(merkleProofArray))
		count++

		if count >= batchSize {
			err = executeBatchInsert(ctx, tx, values, valueStrings)
			if err != nil {
				return err
			}
			values = values[:0]
			valueStrings = valueStrings[:0]
			count = 0
		}
	}

	if count > 0 {
		err = executeBatchInsert(ctx, tx, values, valueStrings)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func executeBatchInsert(ctx context.Context, tx *sql.Tx, values []interface{}, valueStrings []string) error {
	stmt := fmt.Sprintf(`INSERT INTO file_metadata (index, hash, merkle_proof) 
		VALUES %s ON CONFLICT (index) DO NOTHING;`, strings.Join(valueStrings, ","))
	_, err := tx.ExecContext(ctx, stmt, values...)
	return err
}

func byteSlicesToByteaArray(byteSlices [][]byte) [][]byte {
	hexStrings := make([][]byte, len(byteSlices))
	for i, b := range byteSlices {
		hexStrings[i] = []byte(fmt.Sprintf(`'%x'`, b))
	}
	return hexStrings
}
