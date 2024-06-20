package users

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (u *UserService) UserGetCurrent(ctx echo.Context, request gen.UserGetCurrentRequestObject) (gen.UserGetCurrentResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	var hasPass bool

	pass, err := u.config.APIRepository.User().GetUserPassword(user.ID)

	if err != nil && !errors.Is(err, db.ErrNotFound) {
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
		user.ID,
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
