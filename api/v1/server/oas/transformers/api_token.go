package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func ToAPIToken(token *dbsqlc.APIToken) *gen.APIToken {
	res := &gen.APIToken{
		Metadata: *toAPIMetadata(sqlchelpers.UUIDToStr(token.ID), token.CreatedAt.Time, token.UpdatedAt.Time),
	}

	if token.ExpiresAt.Valid {
		res.ExpiresAt = token.ExpiresAt.Time
	}

	if token.Name.Valid {
		res.Name = token.Name.String
	}

	return res
}
