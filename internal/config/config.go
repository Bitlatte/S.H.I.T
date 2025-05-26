package config

type Config struct {
	SiteTitle string `mapstructure:"siteTitle"`
	OutputDir string `mapstructure:"outputDir"`
	BaseURL   string `mapstructure:"baseURL"`
}
