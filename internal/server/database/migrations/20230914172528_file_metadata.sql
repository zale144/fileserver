-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS file_metadata (
    index SERIAL PRIMARY KEY,
    hash BYTEA,
    merkle_proof BYTEA[]
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS file_metadata;
-- +goose StatementEnd
