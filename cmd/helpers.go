package cmd

import (
	"errors"
	"fmt"

	"github.com/platanist/nest-cli/internal/config"
)

func saveConfig() error {
	path := app.ConfigPath
	if path == "" {
		var err error
		path, err = config.DefaultPath()
		if err != nil {
			return err
		}
	}
	return config.Save(path, app.Config)
}

func resolveOrigin(name string) (string, config.Origin, error) {
	if name == "" {
		name = app.Config.DefaultOrigin
	}
	if name == "" {
		return "", config.Origin{}, errors.New("origin is required and no default origin configured")
	}
	origin, ok := app.Config.Origins[name]
	if !ok {
		return "", config.Origin{}, fmt.Errorf("origin %q not configured", name)
	}

	switch origin.EffectiveMode() {
	case config.ModeMongo:
		if config.ResolveMongoURI(name, origin) == "" {
			return "", config.Origin{}, fmt.Errorf("origin %q is in mongo mode but has empty mongo_uri", name)
		}
	case config.ModeAPI:
		if origin.APIBaseURL == "" {
			return "", config.Origin{}, fmt.Errorf("origin %q is in api mode but has empty api_base_url", name)
		}
	default:
		return "", config.Origin{}, fmt.Errorf("origin %q has invalid mode", name)
	}
	return name, origin, nil
}
