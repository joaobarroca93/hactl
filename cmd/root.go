package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaobarroca93/hactl/client"
	"github.com/joaobarroca93/hactl/filter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	quiet   bool
	plain   bool

	// restClient is shared across all commands.
	restClient *client.Client

	// entityFilter enforces entity visibility rules.
	entityFilter *filter.Filter
)

var rootCmd = &cobra.Command{
	Use:   "hactl",
	Short: "Control Home Assistant from the command line",
	Long: `hactl is a fast, single-binary CLI for Home Assistant.
Built for scripting, AI agents, and developers who prefer the terminal.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config validation for completion commands.
		if cmd.Name() == "completion" || cmd.Parent() != nil && cmd.Parent().Name() == "completion" {
			return nil
		}
		return initClient(cmd.Name())
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/hactl/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress all output except errors")
	rootCmd.PersistentFlags().BoolVar(&plain, "plain", false, "output compact human-readable prose instead of JSON")

	rootCmd.AddCommand(stateCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(automationCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(summaryCmd)
	rootCmd.AddCommand(eventsCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			cfgDir := filepath.Join(home, ".config", "hactl")
			viper.AddConfigPath(cfgDir)
		}
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	// Map env vars explicitly
	viper.BindEnv("hass_url", "HASS_URL")
	viper.BindEnv("hass_token", "HASS_TOKEN")

	viper.SetDefault("hass_url", "http://homeassistant.local:8123")
	viper.SetDefault("filter.mode", "exposed")

	_ = viper.ReadInConfig()
}

// initClient validates config, creates the REST client, and initialises the
// entity filter. Pass cmdName so the filter can skip cache loading for "sync".
func initClient(cmdName string) error {
	token := viper.GetString("hass_token")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: HASS_TOKEN is required. Set it via the HASS_TOKEN environment variable or hass_token in config.yaml")
		os.Exit(1)
	}
	baseURL := viper.GetString("hass_url")
	if baseURL == "" {
		baseURL = "http://homeassistant.local:8123"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	restClient = client.New(baseURL, token)

	skipCache := cmdName == "sync" || cmdName == "expose" || cmdName == "unexpose" || cmdName == "rename"
	initFilter(skipCache)
	return nil
}

// initFilter validates filter.mode and creates the entity filter.
// skipCache skips loading the on-disk cache (used by hactl sync).
func initFilter(skipCache bool) {
	mode := viper.GetString("filter.mode")
	if mode == "" {
		mode = "exposed"
	}
	if mode != "exposed" && mode != "all" {
		fmt.Fprintf(os.Stderr, "error: invalid filter.mode %q: must be \"exposed\" or \"all\"\n", mode)
		os.Exit(1)
	}
	entityFilter = filter.New(mode, skipCache)
}

// getClient returns the shared REST client, initializing it if needed.
func getClient() *client.Client {
	return restClient
}
