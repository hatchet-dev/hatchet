package users

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (u *UserService) UserGetCurrent(ctx echo.Context, request gen.UserGetCurrentRequestObject) (gen.UserGetCurrentResponseObject, error) {
	user := ctx.Get("user").(*dbsqlc.User)
	userId := sqlchelpers.UUIDToStr(user.ID)

	var hasPass bool

	pass, err := u.config.APIRepository.User().GetUserPassword(ctx.Request().Context(), userId)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if pass != nil {
		hasPass = true
	}

	var hashedEmail *string

	if u.config.Pylon.Secret != "" {
		hashedEmail, err = signMessageWithHMAC(user.Email, u.config.Pylon.Secret)

		if err != nil {
			return nil, err
		}
	}

	transformedUser := transformers.ToUser(user, hasPass, hashedEmail)

	u.config.Analytics.Enqueue(
		"user:current",
		userId,
		nil,
		map[string]interface{}{
			"email": user.Email,
			"name":  transformedUser.Name,
		},
	)

	return gen.UserGetCurrent200JSONResponse(
		*transformedUser,
	), nil
}

func signMessageWithHMAC(message, secret string) (*string, error) {
	secretBytes, err := hex.DecodeString(secret)
	if err != nil {
		return nil, errors.New("unable to decode secret")
	}

	h := hmac.New(sha256.New, secretBytes)
	h.Write([]byte(message))
	signature := h.Sum(nil)

	signedMsg := hex.EncodeToString(signature)

	return &signedMsg, nil
}
