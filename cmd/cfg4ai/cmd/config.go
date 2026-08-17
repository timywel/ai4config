package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// config.yaml 结构（ARCHITECTURE §7）。
type appConfig struct {
	AI struct {
		Provider string `yaml:"provider,omitempty"`
		BaseURL  string `yaml:"base_url,omitempty"`
		Model    string `yaml:"model,omitempty"`
		Enabled  bool   `yaml:"enabled,omitempty"`
	} `yaml:"ai,omitempty"`
	Secrets struct {
		Backend string `yaml:"backend,omitempty"` // auto|keyring|file|none
	} `yaml:"secrets,omitempty"`
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "读写 cfg4ai 自身配置（config.yaml）",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置配置项（如 ai.provider / secrets.backend）",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		cfg := loadAppConfig(repo)
		key, val := args[0], args[1]
		switch key {
		case "ai.provider":
			cfg.AI.Provider = val
		case "ai.base_url":
			cfg.AI.BaseURL = val
		case "ai.model":
			cfg.AI.Model = val
		case "ai.enabled":
			cfg.AI.Enabled = val == "true"
		case "secrets.backend":
			cfg.Secrets.Backend = val
		default:
			return exitErr(2, "未知配置键 %q", key)
		}
		if err := saveAppConfig(repo, cfg); err != nil {
			return exitErr(1, "写入配置失败: %v", err)
		}
		fmt.Printf("已设置 %s\n", key)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出全部配置",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := openRepo()
		if err != nil {
			return exitErr(1, "打开仓库失败: %v", err)
		}
		cfg := loadAppConfig(repo)
		output(cfg)
		if !isJSON() {
			data, _ := yaml.Marshal(cfg)
			fmt.Print(string(data))
		}
		return nil
	},
}

func loadAppConfig(repo interface{ Path(...string) string }) *appConfig {
	cfg := &appConfig{}
	data, err := os.ReadFile(repo.Path("config.yaml"))
	if err != nil {
		return cfg
	}
	yaml.Unmarshal(data, cfg)
	return cfg
}

func saveAppConfig(repo interface{ Path(...string) string }, cfg *appConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(repo.Path("config.yaml"), data, 0o600)
}

func init() {
	configCmd.AddCommand(configSetCmd, configListCmd)
	rootCmd.AddCommand(configCmd)
}
