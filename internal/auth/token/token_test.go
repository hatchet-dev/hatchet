//go:build integration

package token_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/auth/token"
	"github.com/hatchet-dev/hatchet/internal/config/database"
	"github.com/hatchet-dev/hatchet/internal/encryption"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestCreateTenantToken(t *testing.T) {
	testutils.RunTestWithDatabase(t, func(conf *database.Config) error {
		jwtManager := getJWTManager(t, conf)

		tenantId := uuid.New().String()

		// create the tenant
		slugSuffix, err := encryption.GenerateRandomBytes(8)

		if err != nil {
			t.Fatal(err.Error())
		}

		_, err = conf.Repository.Tenant().CreateTenant(&repository.CreateTenantOpts{
			ID:   &tenantId,
			Name: "test-tenant",
			Slug: fmt.Sprintf("test-tenant-%s", slugSuffix),
		})

		if err != nil {
			t.Fatal(err.Error())
		}

		token, err := jwtManager.GenerateTenantToken(tenantId, "test token")

		if err != nil {
			t.Fatal(err.Error())
		}

		// validate the token
		newTenantId, err := jwtManager.ValidateTenantToken(token)

		assert.NoError(t, err)
		assert.Equal(t, tenantId, newTenantId)

		return nil
	})
}

func TestRevokeTenantToken(t *testing.T) {
	testutils.RunTestWithDatabase(t, func(conf *database.Config) error {
		jwtManager := getJWTManager(t, conf)

		tenantId := uuid.New().String()

		// create the tenant
		slugSuffix, err := encryption.GenerateRandomBytes(8)

		if err != nil {
			t.Fatal(err.Error())
		}

		_, err = conf.Repository.Tenant().CreateTenant(&repository.CreateTenantOpts{
			ID:   &tenantId,
			Name: "test-tenant",
			Slug: fmt.Sprintf("test-tenant-%s", slugSuffix),
		})

		if err != nil {
			t.Fatal(err.Error())
		}

		token, err := jwtManager.GenerateTenantToken(tenantId, "test token")

		if err != nil {
			t.Fatal(err.Error())
		}

		// validate the token
		_, err = jwtManager.ValidateTenantToken(token)

		assert.NoError(t, err)

		// revoke the token
		apiTokens, err := conf.Repository.APIToken().ListAPITokensByTenant(tenantId)

		if err != nil {
			t.Fatal(err.Error())
		}

		assert.Len(t, apiTokens, 1)
		err = conf.Repository.APIToken().RevokeAPIToken(apiTokens[0].ID)

		if err != nil {
			t.Fatal(err.Error())
		}

		// validate the token again
		_, err = jwtManager.ValidateTenantToken(token)

		assert.Error(t, err)

		return nil
	})
}

func getJWTManager(t *testing.T, conf *database.Config) token.JWTManager {
	t.Helper()

	masterKeyBytes, privateJWTBytes, publicJWTBytes, err := encryption.GenerateLocalKeys()

	if err != nil {
		t.Fatal(err.Error())
	}

	encryptionService, err := encryption.NewLocalEncryption(masterKeyBytes, privateJWTBytes, publicJWTBytes)

	if err != nil {
		t.Fatal(err.Error())
	}

	tokenRepo := conf.Repository.APIToken()

	jwtManager, err := token.NewJWTManager(encryptionService, tokenRepo, &token.TokenOpts{
		Issuer:   "hatchet",
		Audience: "hatchet",
	})

	if err != nil {
		t.Fatal(err.Error())
	}

	return jwtManager
}
