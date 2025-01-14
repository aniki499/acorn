package cli

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/acorn/pkg/version"
	bversion "github.com/acorn-io/baaah/pkg/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewInfo(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Info{client: c.ClientFactory}, cobra.Command{
		Use:          "info",
		SilenceUsage: true,
		Short:        "Info about acorn installation",
		Args:         cobra.NoArgs,
	})
	return cmd
}

type Info struct {
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o" default:"yaml"`
	client ClientFactory
}

type ClientServerVersion struct {
	Client struct {
		Version bversion.Version  `json:"version,omitempty"`
		CLI     *config.CLIConfig `json:"cli,omitempty"`
	} `json:"client,omitempty"`
	Server  apiv1.InfoSpec `json:"server,omitempty"`
	Project struct {
		PublicKeys []apiv1.EncryptionKey `json:"publicKeys,omitempty"`
	} `json:"project,omitempty"`
}

func (s *Info) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	info, err := c.Info(cmd.Context())
	if err != nil {
		return err
	}

	cfg, err := config.ReadCLIConfig()
	if err != nil {
		logrus.Errorf("failed to read CLI config: %v", err)
		cfg = nil
	}

	//Formatting...
	ns := struct {
		PublicKeys []apiv1.EncryptionKey `json:"publicKeys,omitempty"`
	}{PublicKeys: info.Spec.PublicKeys}

	info.Spec.PublicKeys = []apiv1.EncryptionKey{}

	out := table.NewWriter(tables.Info, false, s.Output)
	out.Write(ClientServerVersion{
		Client: struct {
			Version bversion.Version  `json:"version,omitempty"`
			CLI     *config.CLIConfig `json:"cli,omitempty"`
		}{
			Version: version.Get(),
			CLI:     cfg.Sanitize(),
		},
		Server:  info.Spec,
		Project: ns,
	})
	return out.Err()
}
