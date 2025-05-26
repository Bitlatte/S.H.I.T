package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Bitlatte/S.H.I.T/internal/config"

	"github.com/spf13/viper"
)

var cfgFile string
var appConfig config.Config

var rootCmd = &cobra.Command{
	Use:   "S.H.I.T",
	Short: "SHIT SSG - Static HTML Is Terrific!",
	Long: `Static HTML Is Terrific (SHIT) is a CLI tool that will take 
your Markdown content, process it, and output a static HTML website.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
}

func initializeConfig(_ *cobra.Command) error {
	v := viper.New()

	v.SetDefault("outputDir", "public")
	v.SetDefault("baseURL", "")
	v.SetDefault("siteTitle", "My Terrific SHIT Site")

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	v.SetEnvPrefix("SHIT")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if cfgFile != "" {
				return fmt.Errorf("config file %s not found in current directory: %w", cfgFile, err)
			}
			fmt.Println("No config file specified or found in current directory. Using default values and/or environment variables for OutputDir and BaseURL.")
		} else {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		fmt.Println("Using config file:", v.ConfigFileUsed())
	}

	if err := v.Unmarshal(&appConfig); err != nil {
		return fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return nil
}
