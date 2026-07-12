package sqlcv1

import "github.com/jackc/pgx/v5/pgtype"

type UUIDRange = pgtype.Range[pgtype.UUID]

// ListActionsForWorkersRow is the canonical (workerId, actionId) row shape
// returned by AssignmentRepository.ListActionsForWorkers. The legacy fallback
// query produces the same shape, so alias it to keep callers stable.
type ListActionsForWorkersRow = ListActionsForWorkersLegacyFallbackRow
