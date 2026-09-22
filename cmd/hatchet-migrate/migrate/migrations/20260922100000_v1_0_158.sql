-- +goose Up
-- +goose StatementBegin
-- The actions of an endpoint whose task asked for an invocation websocket (streams in the
-- healthcheck catalog), namespaced like registered_actions and written with it by the owner,
-- so every process routing the tenant delivers those tasks over the socket.
ALTER TABLE v1_serverless_endpoint
    ADD COLUMN stream_actions TEXT[] NOT NULL DEFAULT '{}';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_serverless_endpoint DROP COLUMN IF EXISTS stream_actions;
-- +goose StatementEnd
