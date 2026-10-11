// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cfg.toml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func TestNewConfig(t *testing.T) {
	cfg, err := NewConfig(write(t, "project_id = \"p\"\nregion = \"us-central1\"\n"))
	require.NoError(t, err)
	assert.Equal(t, "p", cfg.ProjectID)
	assert.Equal(t, "us-central1", cfg.Region)
}

func TestNewConfigInvalid(t *testing.T) {
	_, err := NewConfig(write(t, "region = \"us-central1\"\n"))
	assert.ErrorContains(t, err, "missing project_id")
	_, err = NewConfig(write(t, "project_id = \"p\"\nregion = \"us-central1-a\"\n"))
	assert.ErrorContains(t, err, "invalid region", "a zone is not a region")
	_, err = NewConfig(filepath.Join(t.TempDir(), "missing.toml"))
	assert.Error(t, err)
}
