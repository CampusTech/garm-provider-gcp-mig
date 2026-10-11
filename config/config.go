// SPDX-License-Identifier: Apache-2.0

// Package config holds the provider configuration (TOML file passed by GARM).
package config

import (
	"fmt"
	"regexp"

	"github.com/BurntSushi/toml"
)

// Config of the provider. VM shape (machine types, image, disks, network,
// service account, Spot) is NOT here: it lives in each MIG's instance template
// and instance flexibility policy.
type Config struct {
	// ProjectID where the MIGs live.
	ProjectID string `toml:"project_id" json:"project_id" jsonschema:"description=Project of the managed instance groups."`
	// Region used when a pool's extra specs don't set mig.region.
	Region string `toml:"region" json:"region,omitempty" jsonschema:"description=Default region of the MIGs (overridable per pool with mig.region)."`
	// CredentialsFile is a service account JSON key. Empty = application default credentials.
	CredentialsFile string `toml:"credentials_file" json:"credentials_file,omitempty" jsonschema:"description=Service account JSON key. Empty uses application default credentials (recommended on GCP)."`
}

var regionRegex = regexp.MustCompile(`^[a-z]+-[a-z]+[0-9]+$`)

func NewConfig(cfgFile string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(cfgFile, &cfg); err != nil {
		return nil, fmt.Errorf("error decoding config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("error validating config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.ProjectID == "" {
		return fmt.Errorf("missing project_id")
	}
	if c.Region != "" && !regionRegex.MatchString(c.Region) {
		return fmt.Errorf("invalid region %q (expected e.g. us-central1)", c.Region)
	}
	return nil
}
