package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Bitlatte/S.H.I.T/cmd"
	"github.com/Bitlatte/S.H.I.T/internal/model"
	"gopkg.in/yaml.v2"
)

var site model.SiteData

func loadSiteConfig(filename string) error {
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading config file %s: %w", filename, err)
	}

	err = yaml.Unmarshal(yamlFile, &site.Config)
	if err != nil {
		return fmt.Errorf("error unmarshalling config file %s: %w", filename, err)
	}
	return nil
}

func main() {
	if err := loadSiteConfig("config.yaml"); err != nil {
		log.Fatalf("Error loading site configuration: %v", err)
	}
	cmd.Execute(&site)
}