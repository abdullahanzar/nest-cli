package cmd

import "github.com/platanist/nest-cli/internal/config"

type appContext struct {
	ConfigPath string
	Config     *config.Config
}

var app appContext
