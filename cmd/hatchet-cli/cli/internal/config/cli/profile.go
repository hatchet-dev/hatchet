package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/viper"

	"github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// GetProfiles returns all profiles from the config
func GetProfiles() map[string]cli.Profile {
	profiles := make(map[string]cli.Profile)

	viperMutex.RLock()
	defer viperMutex.RUnlock()

	if ProfilesViperConfig == nil {
		return profiles
	}

	profilesMap := ProfilesViperConfig.GetStringMap("profiles")
	for name := range profilesMap {
		tlsStrategy := ProfilesViperConfig.GetString(fmt.Sprintf("profiles.%s.tlsStrategy", name))
		if tlsStrategy == "" {
			tlsStrategy = "tls"
		}
		profile := cli.Profile{
			TenantId:     ProfilesViperConfig.GetString(fmt.Sprintf("profiles.%s.tenantId", name)),
			Name:         ProfilesViperConfig.GetString(fmt.Sprintf("profiles.%s.name", name)),
			Token:        ProfilesViperConfig.GetString(fmt.Sprintf("profiles.%s.token", name)),
			ExpiresAt:    ProfilesViperConfig.GetTime(fmt.Sprintf("profiles.%s.expiresAt", name)),
			ApiServerURL: ProfilesViperConfig.GetString(fmt.Sprintf("profiles.%s.apiServerURL", name)),
			GrpcHostPort: ProfilesViperConfig.GetString(fmt.Sprintf("profiles.%s.grpcHostPort", name)),
			TLSStrategy:  tlsStrategy,
		}
		profiles[name] = profile
	}

	return profiles
}

// GetProfile returns a specific profile by name
func GetProfile(name string) (*cli.Profile, error) {
	viperMutex.RLock()
	defer viperMutex.RUnlock()

	if ProfilesViperConfig == nil {
		return nil, fmt.Errorf("config not initialized")
	}

	key := fmt.Sprintf("profiles.%s", name)
	if !ProfilesViperConfig.IsSet(key) {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	tlsStrategy := ProfilesViperConfig.GetString(fmt.Sprintf("%s.tlsStrategy", key))
	if tlsStrategy == "" {
		tlsStrategy = "tls"
	}

	profile := &cli.Profile{
		TenantId:     ProfilesViperConfig.GetString(fmt.Sprintf("%s.tenantId", key)),
		Name:         ProfilesViperConfig.GetString(fmt.Sprintf("%s.name", key)),
		Token:        ProfilesViperConfig.GetString(fmt.Sprintf("%s.token", key)),
		ExpiresAt:    ProfilesViperConfig.GetTime(fmt.Sprintf("%s.expiresAt", key)),
		ApiServerURL: ProfilesViperConfig.GetString(fmt.Sprintf("%s.apiServerURL", key)),
		GrpcHostPort: ProfilesViperConfig.GetString(fmt.Sprintf("%s.grpcHostPort", key)),
		TLSStrategy:  tlsStrategy,
	}

	return profile, nil
}

// AddProfile adds a new profile to the config
func AddProfile(name string, profile *cli.Profile) error {
	// Validate required fields
	if profile.TenantId == "" {
		return fmt.Errorf("tenantId is required")
	}
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}
	if profile.Token == "" {
		return fmt.Errorf("token is required")
	}
	if profile.ExpiresAt.IsZero() {
		return fmt.Errorf("expiresAt is required")
	}
	if profile.ApiServerURL == "" {
		return fmt.Errorf("apiServerURL is required")
	}
	if profile.GrpcHostPort == "" {
		return fmt.Errorf("grpcHostPort is required")
	}

	// Set default TLS strategy if not provided
	if profile.TLSStrategy == "" {
		profile.TLSStrategy = "tls"
	}

	// Validate TLS strategy
	if profile.TLSStrategy != "tls" && profile.TLSStrategy != "none" {
		return fmt.Errorf("tlsStrategy must be either 'tls' or 'none'")
	}

	unlock, err := acquireLock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	viperMutex.Lock()
	defer viperMutex.Unlock()

	if ProfilesViperConfig == nil {
		return fmt.Errorf("config not initialized")
	}

	// Reload config to get latest state
	if err := reloadConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	ProfilesViperConfig.Set(fmt.Sprintf("profiles.%s.tenantId", name), profile.TenantId)
	ProfilesViperConfig.Set(fmt.Sprintf("profiles.%s.name", name), profile.Name)
	ProfilesViperConfig.Set(fmt.Sprintf("profiles.%s.token", name), profile.Token)
	ProfilesViperConfig.Set(fmt.Sprintf("profiles.%s.expiresAt", name), profile.ExpiresAt)
	ProfilesViperConfig.Set(fmt.Sprintf("profiles.%s.apiServerURL", name), profile.ApiServerURL)
	ProfilesViperConfig.Set(fmt.Sprintf("profiles.%s.grpcHostPort", name), profile.GrpcHostPort)
	ProfilesViperConfig.Set(fmt.Sprintf("profiles.%s.tlsStrategy", name), profile.TLSStrategy)

	return saveConfig()
}

// RemoveProfile removes a profile from the config
func RemoveProfile(name string) error {
	unlock, err := acquireLock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	viperMutex.Lock()
	defer viperMutex.Unlock()

	if ProfilesViperConfig == nil {
		return fmt.Errorf("config not initialized")
	}

	// Reload config to get latest state
	if err := reloadConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	key := fmt.Sprintf("profiles.%s", name)
	if !ProfilesViperConfig.IsSet(key) {
		return fmt.Errorf("profile '%s' not found", name)
	}

	// Get all current settings
	allSettings := ProfilesViperConfig.AllSettings()

	// Remove the profile from the profiles map
	var updatedProfiles map[string]any
	if profiles, ok := allSettings["profiles"].(map[string]any); ok {
		delete(profiles, name)
		updatedProfiles = profiles
	}

	// Check the default profile (Viper normalizes keys to lowercase)
	var currentDefault string
	var hasDefault bool
	if val, ok := allSettings["defaultprofile"]; ok {
		if strVal, isString := val.(string); isString && strVal != "" {
			currentDefault = strVal
			hasDefault = true
		}
	}

	clearDefault := hasDefault && currentDefault == name

	// Create a fresh Viper instance
	newViper := viper.New()
	newViper.SetConfigType("yaml")

	// Set only the profiles (not the default if it was the removed profile)
	newViper.Set("profiles", updatedProfiles)

	// Only set defaultProfile if it exists and wasn't the removed profile
	if hasDefault && !clearDefault {
		newViper.Set("defaultProfile", currentDefault)
	}

	// Save the config
	profilesFilePath := getProfilesFilePath()

	if err := newViper.WriteConfigAs(profilesFilePath); err != nil {
		return err
	}

	// Reload the config into the new viper instance to ensure consistency
	newViper.SetConfigFile(profilesFilePath)
	if err := newViper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to reload config after removing profile: %w", err)
	}

	// Replace the global with the new instance
	ProfilesViperConfig = newViper

	return nil
}

// UpdateProfile updates an existing profile with new values
// Only non-empty fields will be updated
func UpdateProfile(name string, profile *cli.Profile) error {
	unlock, err := acquireLock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	viperMutex.Lock()
	defer viperMutex.Unlock()

	if ProfilesViperConfig == nil {
		return fmt.Errorf("config not initialized")
	}

	// Reload config to get latest state
	if err := reloadConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	key := fmt.Sprintf("profiles.%s", name)
	if !ProfilesViperConfig.IsSet(key) {
		return fmt.Errorf("profile '%s' not found", name)
	}

	if profile.TenantId != "" {
		ProfilesViperConfig.Set(fmt.Sprintf("%s.tenantId", key), profile.TenantId)
	}
	if profile.Name != "" {
		ProfilesViperConfig.Set(fmt.Sprintf("%s.name", key), profile.Name)
	}
	if profile.Token != "" {
		ProfilesViperConfig.Set(fmt.Sprintf("%s.token", key), profile.Token)
	}
	if !profile.ExpiresAt.IsZero() {
		ProfilesViperConfig.Set(fmt.Sprintf("%s.expiresAt", key), profile.ExpiresAt)
	}
	if profile.ApiServerURL != "" {
		ProfilesViperConfig.Set(fmt.Sprintf("%s.apiServerURL", key), profile.ApiServerURL)
	}
	if profile.GrpcHostPort != "" {
		ProfilesViperConfig.Set(fmt.Sprintf("%s.grpcHostPort", key), profile.GrpcHostPort)
	}
	if profile.TLSStrategy != "" {
		// Validate TLS strategy before updating
		if profile.TLSStrategy != "tls" && profile.TLSStrategy != "none" {
			return fmt.Errorf("tlsStrategy must be either 'tls' or 'none'")
		}
		ProfilesViperConfig.Set(fmt.Sprintf("%s.tlsStrategy", key), profile.TLSStrategy)
	}

	return saveConfig()
}

// ListProfiles returns a list of all profile names
func ListProfiles() []string {
	profiles := GetProfiles()
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetDefaultProfile returns the name of the default profile, or empty string if none is set
func GetDefaultProfile() string {
	viperMutex.RLock()
	defer viperMutex.RUnlock()

	if ProfilesViperConfig == nil {
		return ""
	}

	return ProfilesViperConfig.GetString("defaultProfile")
}

// SetDefaultProfile sets the default profile
func SetDefaultProfile(name string) error {
	unlock, err := acquireLock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	viperMutex.Lock()
	defer viperMutex.Unlock()

	if ProfilesViperConfig == nil {
		return fmt.Errorf("config not initialized")
	}

	// Reload config to get latest state
	if err := reloadConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	// Verify the profile exists
	key := fmt.Sprintf("profiles.%s", name)
	if !ProfilesViperConfig.IsSet(key) {
		return fmt.Errorf("profile '%s' not found", name)
	}

	ProfilesViperConfig.Set("defaultProfile", name)
	return saveConfig()
}

// ClearDefaultProfile clears the default profile setting
func ClearDefaultProfile() error {
	unlock, err := acquireLock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	viperMutex.Lock()
	defer viperMutex.Unlock()

	if ProfilesViperConfig == nil {
		return fmt.Errorf("config not initialized")
	}

	// Reload config to get latest state
	if err := reloadConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	// Only proceed if defaultProfile is actually set
	if !ProfilesViperConfig.IsSet("defaultProfile") {
		return nil
	}

	// Get all profiles (Viper normalizes keys to lowercase)
	allSettings := ProfilesViperConfig.AllSettings()

	// Create a fresh Viper instance with only profiles (no defaultProfile)
	newViper := viper.New()
	newViper.SetConfigType("yaml")

	// Copy only the profiles map, explicitly excluding defaultProfile
	// Note: Viper stores keys as lowercase in AllSettings()
	if profiles, ok := allSettings["profiles"]; ok {
		newViper.Set("profiles", profiles)
	}

	// Save the config
	profilesFilePath := getProfilesFilePath()
	if err := newViper.WriteConfigAs(profilesFilePath); err != nil {
		return err
	}

	// Reload the config into the new viper instance to ensure consistency
	newViper.SetConfigFile(profilesFilePath)
	if err := newViper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to reload config after clearing default: %w", err)
	}

	// Replace the global with the new instance
	ProfilesViperConfig = newViper

	return nil

}

// getProfilesFilePath returns the path to the profiles config file
func getProfilesFilePath() string {
	hatchetDir := filepath.Join(HomeDir, ".hatchet")

	// Get the profile filename from CLIConfig, default to "profiles.yaml"
	profileFileName := "profiles.yaml"
	if CLIConfig != nil && CLIConfig.ProfileFileName != "" {
		profileFileName = CLIConfig.ProfileFileName
	}

	return filepath.Join(hatchetDir, profileFileName)
}

// reloadConfig reloads the config from disk to get latest state
func reloadConfig() error {
	profilesFilePath := getProfilesFilePath()

	// Check if config file exists
	if _, err := os.Stat(profilesFilePath); os.IsNotExist(err) {
		// Config doesn't exist yet, nothing to reload
		return nil
	}

	// Viper caches values set via Set(), so we need to merge file values
	// The safest way is to read the file directly and reset all values
	data, err := os.ReadFile(profilesFilePath)
	if err != nil {
		return err
	}

	// If file is empty, reset to empty profiles
	if len(data) == 0 {
		ProfilesViperConfig.Set("profiles", make(map[string]any))
		return nil
	}

	// Read the config to merge it properly
	if err := ProfilesViperConfig.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

// saveConfig writes the current config to the config file
func saveConfig() error {
	profilesFilePath := getProfilesFilePath()
	return ProfilesViperConfig.WriteConfigAs(profilesFilePath)
}
