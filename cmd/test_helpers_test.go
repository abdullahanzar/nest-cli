package cmd

import (
	"bytes"
	"testing"

	"github.com/platanist/nest-cli/internal/config"
	"github.com/spf13/cobra"
)

func withFreshApp(t *testing.T) {
	t.Helper()
	app = appContext{Config: config.Default()}
	if app.Config.Origins == nil {
		app.Config.Origins = map[string]config.Origin{}
	}
}

func executeForTest(t *testing.T, command *cobra.Command, args ...string) (string, error) {
	t.Helper()

	buf := &bytes.Buffer{}
	command.SetOut(buf)
	command.SetErr(buf)
	command.SetArgs(args)
	err := command.Execute()
	return buf.String(), err
}
