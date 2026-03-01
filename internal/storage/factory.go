package storage

import (
	"fmt"

	"github.com/platanist/nest-cli/internal/config"
)

func NewBackend(originName string, origin config.Origin, token string) (Backend, error) {
	switch origin.EffectiveMode() {
	case config.ModeMongo:
		return newMongoBackend(originName, origin)
	case config.ModeAPI:
		if origin.APIBaseURL == "" {
			return nil, fmt.Errorf("origin %q api mode requires api_base_url", originName)
		}
		return newAPIBackend(origin.APIBaseURL, token), nil
	default:
		return nil, fmt.Errorf("origin %q has invalid mode", originName)
	}
}
