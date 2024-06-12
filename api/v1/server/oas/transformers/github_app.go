package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func ToInstallation(gai *db.GithubAppInstallationModel) *gen.GithubAppInstallation {
	res := &gen.GithubAppInstallation{
		Metadata:    *toAPIMetadata(gai.ID, gai.CreatedAt, gai.UpdatedAt),
		AccountName: gai.AccountName,
	}

	if settingsUrl, ok := gai.InstallationSettingsURL(); ok {
		res.InstallationSettingsUrl = settingsUrl
	}

	if avatarURL, ok := gai.AccountAvatarURL(); ok {
		res.AccountAvatarUrl = avatarURL
	}

	return res
}
