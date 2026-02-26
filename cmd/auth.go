package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaobarroca93/hactl/client"
	"github.com/joaobarroca93/hactl/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	// Override root's PersistentPreRunE so auth subcommands don't require
	// a token to already be configured.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save a long-lived access token to config",
	RunE:  runAuthLogin,
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current authenticated user and HA instance",
	RunE:  runAuthWhoami,
}

var authCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify the token is valid (exit 0 = ok, exit 1 = fail)",
	RunE:  runAuthCheck,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd, authWhoamiCmd, authCheckCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Home Assistant URL [http://homeassistant.local:8123]: ")
	hassURL, _ := reader.ReadString('\n')
	hassURL = strings.TrimSpace(hassURL)
	if hassURL == "" {
		hassURL = "http://homeassistant.local:8123"
	}
	hassURL = strings.TrimRight(hassURL, "/")

	fmt.Printf("\nCreate a token at: %s/profile/security\n", hassURL)
	fmt.Print("Long-lived access token: ")

	var token string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}
		fmt.Println()
		token = strings.TrimSpace(string(b))
	} else {
		t, _ := reader.ReadString('\n')
		token = strings.TrimSpace(t)
	}

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Validate connectivity and auth before saving.
	c := client.New(hassURL, token)
	if err := c.Ping(); err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Fetch version for confirmation message.
	version := ""
	if haCfg, err := c.GetConfig(); err == nil {
		version, _ = haCfg["version"].(string)
	}

	if err := saveAuthConfig(hassURL, token); err != nil {
		return err
	}

	path, _ := authConfigPath()
	if version != "" {
		fmt.Printf("Connected to Home Assistant %s\n", version)
	} else {
		fmt.Println("Connected to Home Assistant")
	}
	fmt.Printf("Config saved to %s\n", path)
	return nil
}

func runAuthWhoami(cmd *cobra.Command, args []string) error {
	c, hassURL, err := newAuthClient()
	if err != nil {
		return err
	}

	haCfg, err := c.GetConfig()
	if err != nil {
		return fmt.Errorf("unauthorized: %w", err)
	}

	version, _ := haCfg["version"].(string)

	if quiet {
		return nil
	}
	if plain {
		output.PrintPlain(fmt.Sprintf("Home Assistant %s · %s", version, hassURL))
		return nil
	}
	return output.PrintJSON(map[string]string{
		"version":  version,
		"hass_url": hassURL,
	})
}

func runAuthCheck(cmd *cobra.Command, args []string) error {
	c, _, err := newAuthClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: not configured:", err)
		os.Exit(1)
	}

	if err := c.Ping(); err != nil {
		fmt.Fprintln(os.Stderr, "error: unauthorized:", err)
		os.Exit(1)
	}

	if !quiet {
		fmt.Println("token valid")
	}
	return nil
}

// newAuthClient builds a REST client from the current viper config without
// requiring the global initClient() to have already run.
func newAuthClient() (*client.Client, string, error) {
	token := viper.GetString("hass_token")
	if token == "" {
		return nil, "", fmt.Errorf("not configured: run hactl auth login")
	}
	baseURL := strings.TrimRight(viper.GetString("hass_url"), "/")
	return client.New(baseURL, token), baseURL, nil
}

// authConfigPath returns the path to the config file. It prefers the path
// viper determined from the current config, falling back to the default location.
func authConfigPath() (string, error) {
	if p := viper.ConfigFileUsed(); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "hactl", "config.yaml"), nil
}

// saveAuthConfig persists hass_url and hass_token to the config file,
// merging with any existing settings so other keys (e.g. filter.mode) are
// preserved.
func saveAuthConfig(hassURL, token string) error {
	path, err := authConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config to preserve other keys.
	existing := make(map[string]any)
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, &existing)
	}
	existing["hass_url"] = hassURL
	existing["hass_token"] = token
	if _, ok := existing["filter"]; !ok {
		existing["filter"] = map[string]any{"mode": "exposed"}
	}

	// Marshal to a node tree so we can attach comments before writing.
	raw, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("failed to parse config for annotation: %w", err)
	}
	annotateFilterMode(&doc)
	data, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("failed to marshal annotated config: %w", err)
	}
	// 0600: token is sensitive, owner-read-only.
	return os.WriteFile(path, data, 0600)
}

// annotateFilterMode adds an inline comment to the filter.mode value explaining
// the valid options.
func annotateFilterMode(doc *yaml.Node) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != "filter" {
			continue
		}
		filterVal := root.Content[i+1]
		if filterVal.Kind != yaml.MappingNode {
			return
		}
		for j := 0; j+1 < len(filterVal.Content); j += 2 {
			if filterVal.Content[j].Value == "mode" {
				filterVal.Content[j+1].LineComment = "# exposed = only Assist-exposed entities; all = every entity"
				return
			}
		}
	}
}
