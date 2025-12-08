-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v1_payload_cutover_queue_item_0_tenant_id_cut_over_at ON v1_payload_cutover_queue_item_0 (tenant_id, cut_over_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v1_payload_cutover_queue_item_1_tenant_id_cut_over_at ON v1_payload_cutover_queue_item_1 (tenant_id, cut_over_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v1_payload_cutover_queue_item_2_tenant_id_cut_over_at ON v1_payload_cutover_queue_item_2 (tenant_id, cut_over_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_v1_payload_cutover_queue_item_3_tenant_id_cut_over_at ON v1_payload_cutover_queue_item_3 (tenant_id, cut_over_at);
CREATE INDEX IF NOT EXISTS idx_v1_payload_cutover_queue_item_tenant_id_cut_over_at ON v1_payload_cutover_queue_item (tenant_id, cut_over_at);

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX IF EXISTS idx_v1_payload_cutover_queue_item_tenant_id_cut_over_at;
