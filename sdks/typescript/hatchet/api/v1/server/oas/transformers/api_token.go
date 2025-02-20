package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func ToAPIToken(token *db.APITokenModel) *gen.APIToken {
	res := &gen.APIToken{
		Metadata: *toAPIMetadata(token.ID, token.CreatedAt, token.UpdatedAt),
	}

	if expiresAt, ok := token.ExpiresAt(); ok {
		res.ExpiresAt = expiresAt
	}

	if name, ok := token.Name(); ok {
		res.Name = name
	}

	return res
}
