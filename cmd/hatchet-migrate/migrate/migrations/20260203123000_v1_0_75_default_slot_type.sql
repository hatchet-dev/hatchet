-- +goose Up
-- +goose StatementBegin
UPDATE v1_worker_slot_capacity
SET slot_type = 'default'
WHERE slot_type = 'legacy';

UPDATE v1_step_slot_requirement
SET slot_type = 'default'
WHERE slot_type = 'legacy';

UPDATE v1_task_runtime_slot
SET slot_type = 'default'
WHERE slot_type = 'legacy';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE v1_worker_slot_capacity
SET slot_type = 'legacy'
WHERE slot_type = 'default';

UPDATE v1_step_slot_requirement
SET slot_type = 'legacy'
WHERE slot_type = 'default';

UPDATE v1_task_runtime_slot
SET slot_type = 'legacy'
WHERE slot_type = 'default';
-- +goose StatementEnd
