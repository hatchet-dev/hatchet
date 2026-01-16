package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestConfig creates a temporary config directory and initializes viper
func setupTestConfig(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory for test config
	tempDir, err := os.MkdirTemp("", "hatchet-test-*")
	require.NoError(t, err)

	hatchetDir := filepath.Join(tempDir, ".hatchet")
	err = os.MkdirAll(hatchetDir, 0755)
	require.NoError(t, err)

	// Store original values
	originalHomeDir := HomeDir
	originalViperConfig := ProfilesViperConfig
	originalCLIConfig := CLIConfig

	// Set up test CLI config
	HomeDir = tempDir
	CLIConfig = &cliconfig.CLIConfig{
		ProfileFileName: "profiles.yaml",
	}

	// Set up test profiles viper config
	profilesFilePath := filepath.Join(hatchetDir, "profiles.yaml")
	ProfilesViperConfig = viper.New()
	ProfilesViperConfig.SetConfigFile(profilesFilePath)
	ProfilesViperConfig.SetConfigType("yaml")

	// Initialize with empty config
	ProfilesViperConfig.Set("profiles", make(map[string]interface{}))

	cleanup := func() {
		HomeDir = originalHomeDir
		ProfilesViperConfig = originalViperConfig
		CLIConfig = originalCLIConfig
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// makeTestProfile creates a valid test profile with the given name and token
// using reasonable defaults for other required fields
func makeTestProfile(name, token string) *cliconfig.Profile {
	expiresAt := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	return &cliconfig.Profile{
		TenantId:     "tenant-123",
		Name:         name,
		Token:        token,
		ExpiresAt:    expiresAt,
		ApiServerURL: "http://localhost:8080",
		GrpcHostPort: "localhost:7077",
	}
}

func TestAddProfile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	expiresAt := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	profile := &cliconfig.Profile{
		TenantId:     "tenant-123",
		Name:         "test-profile",
		Token:        "test-token-123",
		ExpiresAt:    expiresAt,
		ApiServerURL: "http://localhost:8080",
		GrpcHostPort: "localhost:7077",
	}

	err := AddProfile("test-profile", profile)
	require.NoError(t, err)

	retrieved, err := GetProfile("test-profile")
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-token-123", retrieved.Token)
	assert.Equal(t, "tenant-123", retrieved.TenantId)
	assert.Equal(t, "test-profile", retrieved.Name)
	assert.Equal(t, expiresAt, retrieved.ExpiresAt)
	assert.Equal(t, "http://localhost:8080", retrieved.ApiServerURL)
	assert.Equal(t, "localhost:7077", retrieved.GrpcHostPort)
}

func TestAddProfile_NilConfig(t *testing.T) {
	originalViperConfig := ProfilesViperConfig
	ProfilesViperConfig = nil
	defer func() { ProfilesViperConfig = originalViperConfig }()

	expiresAt := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	profile := &cliconfig.Profile{
		TenantId:     "tenant-123",
		Name:         "test-profile",
		Token:        "test-token",
		ExpiresAt:    expiresAt,
		ApiServerURL: "http://localhost:8080",
		GrpcHostPort: "localhost:7077",
	}

	err := AddProfile("test-profile", profile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not initialized")
}

func TestAddProfile_MissingRequiredFields(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	expiresAt := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	testCases := []struct {
		name          string
		profile       *cliconfig.Profile
		expectedError string
	}{
		{
			name: "missing tenantId",
			profile: &cliconfig.Profile{
				Name:         "test",
				Token:        "token",
				ExpiresAt:    expiresAt,
				ApiServerURL: "http://localhost:8080",
				GrpcHostPort: "localhost:7077",
			},
			expectedError: "tenantId is required",
		},
		{
			name: "missing name",
			profile: &cliconfig.Profile{
				TenantId:     "tenant-123",
				Token:        "token",
				ExpiresAt:    expiresAt,
				ApiServerURL: "http://localhost:8080",
				GrpcHostPort: "localhost:7077",
			},
			expectedError: "profile name is required",
		},
		{
			name: "missing token",
			profile: &cliconfig.Profile{
				TenantId:     "tenant-123",
				Name:         "test",
				ExpiresAt:    expiresAt,
				ApiServerURL: "http://localhost:8080",
				GrpcHostPort: "localhost:7077",
			},
			expectedError: "token is required",
		},
		{
			name: "missing expiresAt",
			profile: &cliconfig.Profile{
				TenantId:     "tenant-123",
				Name:         "test",
				Token:        "token",
				ApiServerURL: "http://localhost:8080",
				GrpcHostPort: "localhost:7077",
			},
			expectedError: "expiresAt is required",
		},
		{
			name: "missing apiServerURL",
			profile: &cliconfig.Profile{
				TenantId:     "tenant-123",
				Name:         "test",
				Token:        "token",
				ExpiresAt:    expiresAt,
				GrpcHostPort: "localhost:7077",
			},
			expectedError: "apiServerURL is required",
		},
		{
			name: "missing grpcHostPort",
			profile: &cliconfig.Profile{
				TenantId:     "tenant-123",
				Name:         "test",
				Token:        "token",
				ExpiresAt:    expiresAt,
				ApiServerURL: "http://localhost:8080",
			},
			expectedError: "grpcHostPort is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := AddProfile("test-profile", tc.profile)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

func TestGetProfile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add a profile first
	err := AddProfile("my-profile", makeTestProfile("my-profile", "my-token-456"))
	require.NoError(t, err)

	// Get the profile
	profile, err := GetProfile("my-profile")
	require.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Equal(t, "my-token-456", profile.Token)
	assert.Equal(t, "tenant-123", profile.TenantId)
	assert.Equal(t, "my-profile", profile.Name)

	expectedTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	assert.Equal(t, expectedTime, profile.ExpiresAt)
	assert.Equal(t, "http://localhost:8080", profile.ApiServerURL)
	assert.Equal(t, "localhost:7077", profile.GrpcHostPort)
}

func TestGetProfile_NotFound(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	profile, err := GetProfile("non-existent")
	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetProfile_NilConfig(t *testing.T) {
	originalViperConfig := ProfilesViperConfig
	ProfilesViperConfig = nil
	defer func() { ProfilesViperConfig = originalViperConfig }()

	profile, err := GetProfile("test-profile")
	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Contains(t, err.Error(), "config not initialized")
}

func TestGetProfiles(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Initially should be empty
	profiles := GetProfiles()
	assert.Empty(t, profiles)

	// Add profiles
	err := AddProfile("profile1", makeTestProfile("profile1", "token1"))
	require.NoError(t, err)
	err = AddProfile("profile2", makeTestProfile("profile2", "token2"))
	require.NoError(t, err)
	err = AddProfile("profile3", makeTestProfile("profile3", "token3"))
	require.NoError(t, err)

	// Get all profiles
	profiles = GetProfiles()
	assert.Len(t, profiles, 3)
	assert.Contains(t, profiles, "profile1")
	assert.Contains(t, profiles, "profile2")
	assert.Contains(t, profiles, "profile3")
	assert.Equal(t, "token1", profiles["profile1"].Token)
	assert.Equal(t, "token2", profiles["profile2"].Token)
	assert.Equal(t, "token3", profiles["profile3"].Token)
}

func TestGetProfiles_NilConfig(t *testing.T) {
	originalViperConfig := ProfilesViperConfig
	ProfilesViperConfig = nil
	defer func() { ProfilesViperConfig = originalViperConfig }()

	profiles := GetProfiles()
	assert.Empty(t, profiles)
}

func TestUpdateProfile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add initial profile
	err := AddProfile("update-test", makeTestProfile("update-test", "old-token"))
	require.NoError(t, err)

	// Update token only
	err = UpdateProfile("update-test", &cliconfig.Profile{
		Token: "new-token",
	})
	require.NoError(t, err)

	profile, err := GetProfile("update-test")
	require.NoError(t, err)
	assert.Equal(t, "new-token", profile.Token)
	assert.Equal(t, "http://localhost:8080", profile.ApiServerURL) // Should remain unchanged

	// Update API server URL only
	err = UpdateProfile("update-test", &cliconfig.Profile{
		ApiServerURL: "http://localhost:9090",
	})
	require.NoError(t, err)

	// Verify update persisted
	profile, err = GetProfile("update-test")
	require.NoError(t, err)
	assert.Equal(t, "new-token", profile.Token) // Should remain from previous update
	assert.Equal(t, "http://localhost:9090", profile.ApiServerURL)

	// Update both
	err = UpdateProfile("update-test", &cliconfig.Profile{
		ApiServerURL: "http://localhost:7070",
		Token:        "newest-token",
	})
	require.NoError(t, err)

	profile, err = GetProfile("update-test")
	require.NoError(t, err)
	assert.Equal(t, "newest-token", profile.Token)
	assert.Equal(t, "http://localhost:7070", profile.ApiServerURL)
}

func TestUpdateProfile_NotFound(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	err := UpdateProfile("non-existent", &cliconfig.Profile{
		Token: "token",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateProfile_NilConfig(t *testing.T) {
	originalViperConfig := ProfilesViperConfig
	ProfilesViperConfig = nil
	defer func() { ProfilesViperConfig = originalViperConfig }()

	err := UpdateProfile("test", &cliconfig.Profile{
		Token: "token",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not initialized")
}

func TestRemoveProfile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add profiles
	err := AddProfile("remove1", makeTestProfile("remove1", "token1"))
	require.NoError(t, err)
	err = AddProfile("remove2", makeTestProfile("remove2", "token2"))
	require.NoError(t, err)

	// Remove one profile
	err = RemoveProfile("remove1")
	require.NoError(t, err)

	// Verify it's gone
	_, err = GetProfile("remove1")
	assert.Error(t, err)

	// Verify other profile still exists
	profile, err := GetProfile("remove2")
	require.NoError(t, err)
	assert.Equal(t, "token2", profile.Token)
}

func TestRemoveProfile_NotFound(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	err := RemoveProfile("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveProfile_NilConfig(t *testing.T) {
	originalViperConfig := ProfilesViperConfig
	ProfilesViperConfig = nil
	defer func() { ProfilesViperConfig = originalViperConfig }()

	err := RemoveProfile("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not initialized")
}

func TestListProfiles(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Initially empty
	names := ListProfiles()
	assert.Empty(t, names)

	// Add profiles
	err := AddProfile("alpha", makeTestProfile("alpha", "token1"))
	require.NoError(t, err)
	err = AddProfile("beta", makeTestProfile("beta", "token2"))
	require.NoError(t, err)
	err = AddProfile("gamma", makeTestProfile("gamma", "token3"))
	require.NoError(t, err)

	// List profiles
	names = ListProfiles()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "beta")
	assert.Contains(t, names, "gamma")
}

func TestMultipleProfiles_HappyPath(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add multiple profiles
	profiles := []struct {
		name  string
		token string
	}{
		{"dev", "dev-token"},
		{"staging", "staging-token"},
		{"production", "prod-token"},
		{"local", "local-token"},
	}

	for _, p := range profiles {
		err := AddProfile(p.name, makeTestProfile(p.name, p.token))
		require.NoError(t, err, "Failed to add profile %s", p.name)
	}

	// Verify all profiles exist
	allProfiles := GetProfiles()
	assert.Len(t, allProfiles, len(profiles))

	for _, p := range profiles {
		profile, err := GetProfile(p.name)
		require.NoError(t, err, "Failed to get profile %s", p.name)
		assert.Equal(t, p.token, profile.Token, "Token mismatch for profile %s", p.name)
	}

	// Update one profile
	err := UpdateProfile("staging", &cliconfig.Profile{
		ApiServerURL: "http://new-staging.example.com",
		Token:        "new-staging-token",
	})
	require.NoError(t, err)

	updated, err := GetProfile("staging")
	require.NoError(t, err)
	assert.Equal(t, "new-staging-token", updated.Token)
	assert.Equal(t, "http://new-staging.example.com", updated.ApiServerURL)

	// Remove one profile
	err = RemoveProfile("local")
	require.NoError(t, err)

	allProfiles = GetProfiles()
	assert.Len(t, allProfiles, len(profiles)-1)
	assert.NotContains(t, allProfiles, "local")

	// Verify remaining profiles are intact
	for _, p := range profiles[:3] { // First 3 profiles
		profile, err := GetProfile(p.name)
		require.NoError(t, err, "Profile %s should still exist", p.name)
		assert.NotNil(t, profile)
	}
}

func TestConcurrentWrites_AddProfiles(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	numGoroutines := 20
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Concurrently add different profiles
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			profileName := fmt.Sprintf("profile-%d", id)
			token := fmt.Sprintf("token-%d", id)
			if err := AddProfile(profileName, makeTestProfile(profileName, token)); err != nil {
				errors <- fmt.Errorf("failed to add profile %s: %w", profileName, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}
	if len(errorList) > 0 {
		t.Logf("Encountered %d errors during concurrent writes:", len(errorList))
		for _, err := range errorList {
			t.Log(err)
		}
	}

	// Verify all profiles exist
	profiles := GetProfiles()
	assert.GreaterOrEqual(t, len(profiles), 1, "At least some profiles should have been created")

	// Count how many profiles were successfully created
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		profileName := fmt.Sprintf("profile-%d", i)
		if _, err := GetProfile(profileName); err == nil {
			successCount++
		}
	}
	t.Logf("Successfully created %d/%d profiles", successCount, numGoroutines)
}

func TestConcurrentWrites_MixedOperations(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Pre-populate some profiles
	for i := 0; i < 5; i++ {
		profileName := fmt.Sprintf("initial-%d", i)
		err := AddProfile(profileName, makeTestProfile(profileName, fmt.Sprintf("token-%d", i)))
		require.NoError(t, err)
	}

	numGoroutines := 30
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Mix of add, update, and remove operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			switch id % 3 {
			case 0: // Add new profile
				profileName := fmt.Sprintf("new-profile-%d", id)
				if err := AddProfile(profileName, makeTestProfile(profileName, fmt.Sprintf("token-%d", id))); err != nil {
					errors <- fmt.Errorf("add failed: %w", err)
				}
			case 1: // Update existing profile
				profileName := fmt.Sprintf("initial-%d", id%5)
				if err := UpdateProfile(profileName, &cliconfig.Profile{
					Token: fmt.Sprintf("updated-token-%d", id),
				}); err != nil {
					// Profile might have been removed by another goroutine, that's okay
					if err.Error() != fmt.Sprintf("profile '%s' not found", profileName) {
						errors <- fmt.Errorf("update failed: %w", err)
					}
				}
			case 2: // Remove profile
				profileName := fmt.Sprintf("initial-%d", id%5)
				if err := RemoveProfile(profileName); err != nil {
					// Profile might already be removed, that's okay
					if err.Error() != fmt.Sprintf("profile '%s' not found", profileName) {
						errors <- fmt.Errorf("remove failed: %w", err)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for unexpected errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}
	if len(errorList) > 0 {
		t.Logf("Encountered %d unexpected errors during concurrent operations:", len(errorList))
		for _, err := range errorList {
			t.Log(err)
		}
	}

	// Verify config file still exists and is readable
	profiles := GetProfiles()
	t.Logf("Final profile count: %d", len(profiles))
	assert.NotNil(t, profiles, "Profile map should not be nil after concurrent operations")
}

func TestConcurrentWrites_SameProfile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add initial profile
	err := AddProfile("shared", makeTestProfile("shared", "initial-token"))
	require.NoError(t, err)

	numGoroutines := 15
	var wg sync.WaitGroup

	// Multiple goroutines updating the same profile
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = UpdateProfile("shared", &cliconfig.Profile{
				Token: fmt.Sprintf("token-from-goroutine-%d", id),
			})
		}(i)
	}

	wg.Wait()

	// Verify profile still exists and has one of the tokens
	profile, err := GetProfile("shared")
	require.NoError(t, err)
	assert.NotNil(t, profile)
	assert.NotEmpty(t, profile.Token)
	t.Logf("Final token value: %s", profile.Token)
}

func TestProfilePersistence(t *testing.T) {
	tempDir, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add profiles
	err := AddProfile("persist1", makeTestProfile("persist1", "token1"))
	require.NoError(t, err)
	err = AddProfile("persist2", makeTestProfile("persist2", "token2"))
	require.NoError(t, err)

	// Simulate reload by creating new viper instance
	profilesFilePath := filepath.Join(tempDir, ".hatchet", "profiles.yaml")
	newViper := viper.New()
	newViper.SetConfigFile(profilesFilePath)
	newViper.SetConfigType("yaml")

	err = newViper.ReadInConfig()
	require.NoError(t, err)

	// Temporarily replace global config
	oldConfig := ProfilesViperConfig
	ProfilesViperConfig = newViper

	// Verify profiles persisted
	profiles := GetProfiles()
	assert.Len(t, profiles, 2)
	assert.Contains(t, profiles, "persist1")
	assert.Contains(t, profiles, "persist2")

	profile1, err := GetProfile("persist1")
	require.NoError(t, err)
	assert.Equal(t, "token1", profile1.Token)
	assert.Equal(t, "tenant-123", profile1.TenantId)
	assert.Equal(t, "persist1", profile1.Name)

	profile2, err := GetProfile("persist2")
	require.NoError(t, err)
	assert.Equal(t, "token2", profile2.Token)
	assert.Equal(t, "tenant-123", profile2.TenantId)
	assert.Equal(t, "persist2", profile2.Name)

	ProfilesViperConfig = oldConfig
}

func TestProfileWithSpecialCharacters(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	specialProfiles := []struct {
		name  string
		token string
	}{
		{"profile-with-dashes", "token-123"},
		{"profile_with_underscores", "token_456"},
		{"profile.with.dots", "token.789"},
		{"PROFILE_UPPERCASE", "TOKEN_ABC"},
	}

	for _, p := range specialProfiles {
		err := AddProfile(p.name, makeTestProfile(p.name, p.token))
		require.NoError(t, err, "Failed to add profile with name: %s", p.name)

		profile, err := GetProfile(p.name)
		require.NoError(t, err, "Failed to get profile with name: %s", p.name)
		assert.Equal(t, p.token, profile.Token)
	}

	// Verify all profiles exist
	profiles := GetProfiles()
	assert.Len(t, profiles, len(specialProfiles))
}

func TestLockFileCreationAndCleanup(t *testing.T) {
	tempDir, cleanup := setupTestConfig(t)
	defer cleanup()

	lockFilePath := filepath.Join(tempDir, ".hatchet", "config.lock")

	// Verify lock file doesn't exist initially
	_, err := os.Stat(lockFilePath)
	assert.True(t, os.IsNotExist(err), "Lock file should not exist initially")

	// Add a profile (which will acquire and release lock)
	err = AddProfile("test", makeTestProfile("test", "token"))
	require.NoError(t, err)

	// Verify lock file is cleaned up after operation
	_, err = os.Stat(lockFilePath)
	assert.True(t, os.IsNotExist(err), "Lock file should be cleaned up after operation")
}

func TestLockFileExpiration(t *testing.T) {
	tempDir, cleanup := setupTestConfig(t)
	defer cleanup()

	lockFilePath := filepath.Join(tempDir, ".hatchet", "config.lock")

	// Create a stale lock file
	err := os.MkdirAll(filepath.Join(tempDir, ".hatchet"), 0755)
	require.NoError(t, err)

	f, err := os.Create(lockFilePath)
	require.NoError(t, err)
	f.WriteString(time.Now().Add(-10 * time.Second).Format(time.RFC3339))
	f.Close()

	// Set the file's modification time to the past
	pastTime := time.Now().Add(-10 * time.Second)
	err = os.Chtimes(lockFilePath, pastTime, pastTime)
	require.NoError(t, err)

	// Verify lock file is stale
	stat, err := os.Stat(lockFilePath)
	require.NoError(t, err)
	assert.True(t, time.Since(stat.ModTime()) > 5*time.Second, "Lock file should be stale")

	// Operation should succeed by removing stale lock
	err = AddProfile("test", makeTestProfile("test", "token"))
	require.NoError(t, err)

	// Verify profile was created successfully
	profile, err := GetProfile("test")
	require.NoError(t, err)
	assert.Equal(t, "token", profile.Token)
}

func TestConcurrentOperationsWithLock(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	numGoroutines := 50
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errorCount atomic.Int32

	// Pre-populate a profile for updates
	err := AddProfile("target", makeTestProfile("target", "initial-token"))
	require.NoError(t, err)

	// Many goroutines trying to update the same profile
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := UpdateProfile("target", &cliconfig.Profile{
				Token: fmt.Sprintf("token-%d", id),
			})
			if err == nil {
				successCount.Add(1)
			} else {
				errorCount.Add(1)
				t.Logf("Update %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Successful updates: %d, Failed updates: %d", successCount.Load(), errorCount.Load())

	// Profile should exist with one of the tokens
	profile, err := GetProfile("target")
	require.NoError(t, err)
	assert.NotNil(t, profile)
	assert.Contains(t, profile.Token, "token-")
	t.Logf("Final token: %s", profile.Token)

	// All operations should have succeeded (lock mechanism should prevent failures)
	assert.Equal(t, int32(numGoroutines), successCount.Load(), "All updates should succeed with proper locking")
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add initial profiles
	for i := 0; i < 5; i++ {
		profileName := fmt.Sprintf("profile-%d", i)
		err := AddProfile(profileName, makeTestProfile(profileName, fmt.Sprintf("token-%d", i)))
		require.NoError(t, err)
	}

	numReaders := 20
	numWriters := 10
	var wg sync.WaitGroup

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				profiles := GetProfiles()
				assert.NotNil(t, profiles)
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			profileName := fmt.Sprintf("profile-%d", id%5)
			for j := 0; j < 3; j++ {
				err := UpdateProfile(profileName, &cliconfig.Profile{
					Token: fmt.Sprintf("updated-token-%d-%d", id, j),
				})
				if err != nil {
					t.Logf("Writer %d failed: %v", id, err)
				}
				time.Sleep(20 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify all profiles still exist and are valid
	profiles := GetProfiles()
	assert.GreaterOrEqual(t, len(profiles), 5, "All profiles should still exist")

	for i := 0; i < 5; i++ {
		profileName := fmt.Sprintf("profile-%d", i)
		profile, err := GetProfile(profileName)
		require.NoError(t, err, "Profile %s should exist", profileName)
		assert.NotEmpty(t, profile.Token)
	}
}

func TestGetDefaultProfile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Initially, no default should be set
	defaultProfile := GetDefaultProfile()
	assert.Empty(t, defaultProfile)
}

func TestSetDefaultProfile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add a profile
	err := AddProfile("test-profile", makeTestProfile("test-profile", "token-123"))
	require.NoError(t, err)

	// Set it as default
	err = SetDefaultProfile("test-profile")
	require.NoError(t, err)

	// Verify it's set as default
	defaultProfile := GetDefaultProfile()
	assert.Equal(t, "test-profile", defaultProfile)
}

func TestSetDefaultProfile_NotFound(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Try to set a non-existent profile as default
	err := SetDefaultProfile("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Default should still be empty
	defaultProfile := GetDefaultProfile()
	assert.Empty(t, defaultProfile)
}

func TestSetDefaultProfile_NilConfig(t *testing.T) {
	originalViperConfig := ProfilesViperConfig
	ProfilesViperConfig = nil
	defer func() { ProfilesViperConfig = originalViperConfig }()

	err := SetDefaultProfile("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not initialized")
}

func TestClearDefaultProfile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add a profile and set it as default
	err := AddProfile("test-profile", makeTestProfile("test-profile", "token-123"))
	require.NoError(t, err)
	err = SetDefaultProfile("test-profile")
	require.NoError(t, err)

	// Verify it's set
	defaultProfile := GetDefaultProfile()
	assert.Equal(t, "test-profile", defaultProfile)

	// Clear the default
	err = ClearDefaultProfile()
	require.NoError(t, err)

	// Verify it's cleared
	defaultProfile = GetDefaultProfile()
	assert.Empty(t, defaultProfile)
}

func TestClearDefaultProfile_NilConfig(t *testing.T) {
	originalViperConfig := ProfilesViperConfig
	ProfilesViperConfig = nil
	defer func() { ProfilesViperConfig = originalViperConfig }()

	err := ClearDefaultProfile()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not initialized")
}

func TestRemoveProfile_ClearsDefaultIfRemoved(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add profiles
	err := AddProfile("profile1", makeTestProfile("profile1", "token1"))
	require.NoError(t, err)
	err = AddProfile("profile2", makeTestProfile("profile2", "token2"))
	require.NoError(t, err)

	// Set profile1 as default
	err = SetDefaultProfile("profile1")
	require.NoError(t, err)

	// Verify it's set
	defaultProfile := GetDefaultProfile()
	assert.Equal(t, "profile1", defaultProfile)

	// Remove profile1
	err = RemoveProfile("profile1")
	require.NoError(t, err)

	// Verify default is cleared
	defaultProfile = GetDefaultProfile()
	assert.Empty(t, defaultProfile)

	// Verify profile2 still exists
	profile, err := GetProfile("profile2")
	require.NoError(t, err)
	assert.Equal(t, "token2", profile.Token)
}

func TestRemoveProfile_DoesNotClearDifferentDefault(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add profiles
	err := AddProfile("profile1", makeTestProfile("profile1", "token1"))
	require.NoError(t, err)
	err = AddProfile("profile2", makeTestProfile("profile2", "token2"))
	require.NoError(t, err)

	// Set profile1 as default
	err = SetDefaultProfile("profile1")
	require.NoError(t, err)

	// Remove profile2 (not the default)
	err = RemoveProfile("profile2")
	require.NoError(t, err)

	// Verify default is still profile1
	defaultProfile := GetDefaultProfile()
	assert.Equal(t, "profile1", defaultProfile)
}

func TestDefaultProfilePersistence(t *testing.T) {
	tempDir, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add a profile and set it as default
	err := AddProfile("persist-default", makeTestProfile("persist-default", "token-123"))
	require.NoError(t, err)
	err = SetDefaultProfile("persist-default")
	require.NoError(t, err)

	// Simulate reload by creating new viper instance
	profilesFilePath := filepath.Join(tempDir, ".hatchet", "profiles.yaml")
	newViper := viper.New()
	newViper.SetConfigFile(profilesFilePath)
	newViper.SetConfigType("yaml")

	err = newViper.ReadInConfig()
	require.NoError(t, err)

	// Temporarily replace global config
	oldConfig := ProfilesViperConfig
	ProfilesViperConfig = newViper

	// Verify default profile persisted
	defaultProfile := GetDefaultProfile()
	assert.Equal(t, "persist-default", defaultProfile)

	// Verify profile still exists
	profile, err := GetProfile("persist-default")
	require.NoError(t, err)
	assert.Equal(t, "token-123", profile.Token)

	ProfilesViperConfig = oldConfig
}

func TestSetDefaultProfile_SwitchBetweenProfiles(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add multiple profiles
	err := AddProfile("dev", makeTestProfile("dev", "dev-token"))
	require.NoError(t, err)
	err = AddProfile("staging", makeTestProfile("staging", "staging-token"))
	require.NoError(t, err)
	err = AddProfile("prod", makeTestProfile("prod", "prod-token"))
	require.NoError(t, err)

	// Set dev as default
	err = SetDefaultProfile("dev")
	require.NoError(t, err)
	assert.Equal(t, "dev", GetDefaultProfile())

	// Switch to staging
	err = SetDefaultProfile("staging")
	require.NoError(t, err)
	assert.Equal(t, "staging", GetDefaultProfile())

	// Switch to prod
	err = SetDefaultProfile("prod")
	require.NoError(t, err)
	assert.Equal(t, "prod", GetDefaultProfile())

	// Switch back to dev
	err = SetDefaultProfile("dev")
	require.NoError(t, err)
	assert.Equal(t, "dev", GetDefaultProfile())
}

func TestConcurrentDefaultProfileOperations(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Add profiles
	for i := 0; i < 5; i++ {
		profileName := fmt.Sprintf("profile-%d", i)
		err := AddProfile(profileName, makeTestProfile(profileName, fmt.Sprintf("token-%d", i)))
		require.NoError(t, err)
	}

	numGoroutines := 20
	var wg sync.WaitGroup

	// Multiple goroutines setting different defaults
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			profileName := fmt.Sprintf("profile-%d", id%5)
			_ = SetDefaultProfile(profileName)
		}(i)
	}

	wg.Wait()

	// Verify a default is set and is one of the valid profiles
	defaultProfile := GetDefaultProfile()
	assert.NotEmpty(t, defaultProfile)
	assert.Contains(t, defaultProfile, "profile-")

	// Verify the default profile exists
	profile, err := GetProfile(defaultProfile)
	require.NoError(t, err)
	assert.NotNil(t, profile)
}
