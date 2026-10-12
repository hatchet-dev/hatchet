//go:build integration

package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestUpdateUserFromOAuth(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		ctx := context.Background()

		newUserWithSession := func(t *testing.T, verified bool) (*sqlcv1.User, uuid.UUID) {
			t.Helper()

			hash, err := v1.HashPassword("password-for-tests")
			require.NoError(t, err)

			user, err := conf.V1.User().CreateUser(ctx, &v1.CreateUserOpts{
				Email:         fmt.Sprintf("oauth-%s@example.com", uuid.NewString()),
				EmailVerified: v1.BoolPtr(verified),
				Password:      hash,
			})
			require.NoError(t, err)

			sessionID := uuid.New()
			_, err = conf.V1.UserSession().Create(ctx, &v1.CreateSessionOpts{
				ID:        sessionID,
				ExpiresAt: time.Now().Add(time.Hour),
				UserId:    &user.ID,
				Data:      []byte("{}"),
			})
			require.NoError(t, err)

			return user, sessionID
		}

		oauthLogin := func(verified bool) *v1.UpdateUserOpts {
			return &v1.UpdateUserOpts{
				EmailVerified: v1.BoolPtr(verified),
				OAuth: &v1.OAuthOpts{
					Provider:       "google",
					ProviderUserId: uuid.NewString(),
					AccessToken:    []byte("access-token"),
				},
			}
		}

		t.Run("first verification deletes password and sessions", func(t *testing.T) {
			user, sessionID := newUserWithSession(t, false)

			updated, err := conf.V1.User().UpdateUserFromOAuth(ctx, user.ID, oauthLogin(true))
			require.NoError(t, err)
			assert.True(t, updated.EmailVerified)

			_, err = conf.V1.User().GetUserPassword(ctx, user.ID)
			assert.True(t, errors.Is(err, pgx.ErrNoRows), "expected password to be deleted, got %v", err)

			_, err = conf.V1.UserSession().GetById(ctx, sessionID)
			assert.True(t, errors.Is(err, pgx.ErrNoRows), "expected session to be deleted, got %v", err)
		})

		t.Run("already verified user keeps password and sessions", func(t *testing.T) {
			user, sessionID := newUserWithSession(t, true)

			_, err := conf.V1.User().UpdateUserFromOAuth(ctx, user.ID, oauthLogin(true))
			require.NoError(t, err)

			_, err = conf.V1.User().GetUserPassword(ctx, user.ID)
			assert.NoError(t, err)

			_, err = conf.V1.UserSession().GetById(ctx, sessionID)
			assert.NoError(t, err)
		})

		t.Run("login that does not verify keeps password and sessions", func(t *testing.T) {
			user, sessionID := newUserWithSession(t, false)

			updated, err := conf.V1.User().UpdateUserFromOAuth(ctx, user.ID, oauthLogin(false))
			require.NoError(t, err)
			assert.False(t, updated.EmailVerified)

			_, err = conf.V1.User().GetUserPassword(ctx, user.ID)
			assert.NoError(t, err)

			_, err = conf.V1.UserSession().GetById(ctx, sessionID)
			assert.NoError(t, err)
		})

		return nil
	})
}
