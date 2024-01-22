package loaderutils

import (
	"bytes"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/spf13/viper"
)

func LoadConfigFromViper(bindFunc func(v *viper.Viper), configFile interface{}, files ...[]byte) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	bindFunc(v)

	for _, f := range files {
		err := v.MergeConfig(bytes.NewBuffer(f))

		if err != nil {
			return nil, fmt.Errorf("could not load viper config: %w", err)
		}
	}

	if err := defaults.Set(configFile); err != nil {
		return nil, fmt.Errorf("could not set defaults for config: %w", err)
	}

	err := v.Unmarshal(configFile)

	if err != nil {
		return nil, fmt.Errorf("could not unmarshal viper config: %w", err)
	}

	return v, nil
}
