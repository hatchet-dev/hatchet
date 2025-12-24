-- +goose Up
-- +goose StatementBegin

ALTER TYPE v1_event_type_olap ADD VALUE 'COULD_NOT_SEND_TO_WORKER';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TYPE v1_event_type_olap DROP VALUE 'COULD_NOT_SEND_TO_WORKER';

-- +goose StatementEnd
