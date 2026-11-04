package cmd

import (
	"bytes"
	"testing"

	"github.com/mocyuto/zgt/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigCmd_Raw(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // 実stateの読み込み防止
	// Set up dummy config with placeholders
	config.AppConfig = config.Config{
		Env: map[string]string{
			"PROJECT": "zgt-{{.Repo}}",
		},
	}
	configRaw = true
	defer func() { configRaw = false }()

	b := bytes.NewBufferString("")
	configCmd.SetOut(b)
	rootCmd.SetArgs([]string{"config", "--raw"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	out := b.String()
	assert.Contains(t, out, "--- Merged Configuration (Raw) ---")
	assert.Contains(t, out, "PROJECT: zgt-{{.Repo}}")
}

func TestConfigCmd_Replaced(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // 実stateの読み込み防止
	// Set up dummy config with placeholders
	config.AppConfig = config.Config{
		Env: map[string]string{
			"PROJECT": "zgt-{{.Repo}}",
		},
	}
	configRaw = false

	b := bytes.NewBufferString("")
	configCmd.SetOut(b)
	rootCmd.SetArgs([]string{"config"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	out := b.String()
	assert.Contains(t, out, "--- Merged Configuration (Placeholders Replaced) ---")
	// Since we are not in a real repo during test, .Repo might be empty or something else
	// but it shouldn't be the raw placeholder.
	assert.NotContains(t, out, "PROJECT: zgt-{{.Repo}}")
}
