//go:build integration

package token_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestCreateTenantToken(t *testing.T) { // make sure no cache is used for tests
	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		jwtManager := getJWTManager(t, conf)

		tenantId := uuid.New().String()

		// create the tenant
		slugSuffix, err := random.Generate(8)

		if err != nil {
			t.Fatal(err.Error())
		}

		_, err = conf.APIRepository.Tenant().CreateTenant(&repository.CreateTenantOpts{
			ID:   &tenantId,
			Name: "test-tenant",
			Slug: fmt.Sprintf("test-tenant-%s", slugSuffix),
		})

		if err != nil {
			t.Fatal(err.Error())
		}

		token, err := jwtManager.GenerateTenantToken(context.Background(), tenantId, "test token", false, nil)

		if err != nil {
			t.Fatal(err.Error())
		}

		// validate the token
		newTenantId, _, err := jwtManager.ValidateTenantToken(context.Background(), token.Token)

		assert.NoError(t, err)
		assert.Equal(t, tenantId, newTenantId)

		return nil
	})
}

func TestRevokeTenantToken(t *testing.T) {
	_ = os.Setenv("CACHE_DURATION", "0")

	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		jwtManager := getJWTManager(t, conf)

		tenantId := uuid.New().String()

		// create the tenant
		slugSuffix, err := random.Generate(8)

		if err != nil {
			t.Fatal(err.Error())
		}

		_, err = conf.APIRepository.Tenant().CreateTenant(&repository.CreateTenantOpts{
			ID:   &tenantId,
			Name: "test-tenant",
			Slug: fmt.Sprintf("test-tenant-%s", slugSuffix),
		})

		if err != nil {
			t.Fatal(err.Error())
		}

		token, err := jwtManager.GenerateTenantToken(context.Background(), tenantId, "test token", false, nil)

		if err != nil {
			t.Fatal(err.Error())
		}

		// validate the token
		_, _, err = jwtManager.ValidateTenantToken(context.Background(), token.Token)

		assert.NoError(t, err)

		// revoke the token
		apiTokens, err := conf.APIRepository.APIToken().ListAPITokensByTenant(tenantId)

		if err != nil {
			t.Fatal(err.Error())
		}

		assert.Len(t, apiTokens, 1)
		err = conf.APIRepository.APIToken().RevokeAPIToken(apiTokens[0].ID)

		if err != nil {
			t.Fatal(err.Error())
		}

		// validate the token again
		_, _, err = jwtManager.ValidateTenantToken(context.Background(), token.Token)

		// error as the token was revoked
		assert.Error(t, err)

		return nil
	})
}

func TestRevokeTenantTokenCache(t *testing.T) {
	_ = os.Setenv("CACHE_DURATION", "60s")

	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		jwtManager := getJWTManager(t, conf)

		tenantId := uuid.New().String()

		// create the tenant
		slugSuffix, err := random.Generate(8)

		if err != nil {
			t.Fatal(err.Error())
		}

		_, err = conf.APIRepository.Tenant().CreateTenant(&repository.CreateTenantOpts{
			ID:   &tenantId,
			Name: "test-tenant",
			Slug: fmt.Sprintf("test-tenant-%s", slugSuffix),
		})

		if err != nil {
			t.Fatal(err.Error())
		}

		token, err := jwtManager.GenerateTenantToken(context.Background(), tenantId, "test token", false, nil)

		if err != nil {
			t.Fatal(err.Error())
		}

		// validate the token
		_, _, err = jwtManager.ValidateTenantToken(context.Background(), token.Token)

		assert.NoError(t, err)

		// revoke the token
		apiTokens, err := conf.APIRepository.APIToken().ListAPITokensByTenant(tenantId)

		if err != nil {
			t.Fatal(err.Error())
		}

		assert.Len(t, apiTokens, 1)
		err = conf.APIRepository.APIToken().RevokeAPIToken(apiTokens[0].ID)

		if err != nil {
			t.Fatal(err.Error())
		}

		// validate the token again
		_, _, err = jwtManager.ValidateTenantToken(context.Background(), token.Token)

		// no error as it is cached
		assert.NoError(t, err)

		return nil
	})
}

func getJWTManager(t *testing.T, conf *database.Layer) token.JWTManager {
	t.Helper()

	masterKeyBytes, privateJWTBytes, publicJWTBytes, err := encryption.GenerateLocalKeys()

	if err != nil {
		t.Fatal(err.Error())
	}

	encryptionService, err := encryption.NewLocalEncryption(masterKeyBytes, privateJWTBytes, publicJWTBytes)

	if err != nil {
		t.Fatal(err.Error())
	}

	tokenRepo := conf.EngineRepository.APIToken()

	jwtManager, err := token.NewJWTManager(encryptionService, tokenRepo, &token.TokenOpts{
		Issuer:   "hatchet",
		Audience: "hatchet",
	})

	if err != nil {
		t.Fatal(err.Error())
	}

	return jwtManager
}
