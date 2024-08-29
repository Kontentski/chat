-- +goose Up
-- +goose StatementBegin
ALTER TABLE messages ADD COLUMN type VARCHAR(50);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE messages DELETE COLUMN type;
-- +goose StatementEnd
