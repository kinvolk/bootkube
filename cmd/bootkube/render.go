package main

import (
	"fmt"
	"os"
	"plugin"

	bootkubeplugin "github.com/kubernetes-sigs/bootkube/pkg/plugin"
	"github.com/spf13/cobra"
)

var (
	cmdRender = &cobra.Command{
		Use:          "render",
		Short:        "Render cluster manifests using the specified plugin",
		RunE:         runCmdRender,
		SilenceUsage: true,
	}

	renderOpts struct {
		assetDir    string
		plugin      string
		pluginFlags []string
	}

	pluginOpts bootkubeplugin.Options
)

func init() {
	cmdRoot.AddCommand(cmdRender)

	cmdRender.Flags().StringVar(&pluginOpts.AssetDir, "asset-dir", "", "Output path for rendered assets")
	cmdRender.Flags().StringVar(&renderOpts.plugin, "plugin", "", "Path to the render plugin")
	cmdRender.Flags().StringSliceVar(&renderOpts.pluginFlags, "plugin-flag", []string{}, "The flags to pass to the render plugin")

	cobra.MarkFlagRequired(cmdRender.Flags(), "asset-dir")
	cobra.MarkFlagRequired(cmdRender.Flags(), "plugin")
}

func runCmdRender(cmd *cobra.Command, args []string) error {
	plug, err := plugin.Open(renderOpts.plugin)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	symbol, err := plug.Lookup("Renderer")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var renderer bootkubeplugin.Renderer
	renderer, ok := symbol.(bootkubeplugin.Renderer)
	if !ok {
		fmt.Println("unexpected type from plugin")
		os.Exit(1)
	}

	return renderer.Render(&pluginOpts, renderOpts.pluginFlags)
}
