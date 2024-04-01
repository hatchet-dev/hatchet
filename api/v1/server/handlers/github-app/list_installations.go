package githubapp

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (g *GithubAppService) GithubAppListInstallations(ctx echo.Context, req gen.GithubAppListInstallationsRequestObject) (gen.GithubAppListInstallationsResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	gais, err := g.config.APIRepository.Github().ListGithubAppInstallationsByUserID(user.ID)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.GithubAppInstallation, 0)

	for i := range gais {
		rows = append(rows, *transformers.ToInstallation(&gais[i]))
	}

	return gen.GithubAppListInstallations200JSONResponse(
		gen.ListGithubAppInstallationsResponse{
			Rows: rows,
		},
	), nil
}
