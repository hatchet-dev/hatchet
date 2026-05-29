package sqlcv1

import "github.com/jackc/pgx/v5/pgtype"

type UUIDRange = pgtype.Range[pgtype.UUID]
