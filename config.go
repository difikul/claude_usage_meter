package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type OAuthCredentials struct {
	RateLimitTier string `json:"rateLimitTier"`
	AccessToken   string `json:"accessToken"`
}

type CredentialsFile struct {
	ClaudeAiOauth *OAuthCredentials `json:"claudeAiOauth"`
}

type BudgetOverrides struct {
	FiveHour     *float64 `json:"five_hour"`
	Weekly       *float64 `json:"weekly"`
	WeeklySonnet *float64 `json:"weekly_sonnet"`
}

type AppConfig struct {
	BudgetOverrides *BudgetOverrides `json:"budget_overrides"`
}

type ResolvedBudgets struct {
	FiveHour     float64 `json:"five_hour"`
	Weekly       float64 `json:"weekly"`
	WeeklySonnet float64 `json:"weekly_sonnet"`
}

func claudeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

func readTier() string {
	dir := claudeDir()
	if dir == "" {
		return "unknown"
	}
	data, err := os.ReadFile(filepath.Join(dir, ".credentials.json"))
	if err != nil {
		return "unknown"
	}
	var creds CredentialsFile
	if err := json.Unmarshal(data, &creds); err != nil {
		return "unknown"
	}
	if creds.ClaudeAiOauth != nil && creds.ClaudeAiOauth.RateLimitTier != "" {
		return creds.ClaudeAiOauth.RateLimitTier
	}
	return "unknown"
}

func readOAuthToken() string {
	dir := claudeDir()
	if dir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, ".credentials.json"))
	if err != nil {
		return ""
	}
	var creds CredentialsFile
	if err := json.Unmarshal(data, &creds); err != nil {
		return ""
	}
	if creds.ClaudeAiOauth != nil {
		return creds.ClaudeAiOauth.AccessToken
	}
	return ""
}

func readAppConfig() AppConfig {
	dir := claudeDir()
	if dir == "" {
		return AppConfig{}
	}
	data, err := os.ReadFile(filepath.Join(dir, "usage-meter-config.json"))
	if err != nil {
		return AppConfig{}
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}
	}
	return cfg
}

func tierDefaults(tier string) (float64, float64, float64) {
	switch tier {
	case "default_claude_max_5x":
		return 93.0, 1090.0, 160.0
	case "default_claude_max_20x":
		return 372.0, 4360.0, 640.0
	case "default_claude_pro":
		return 18.6, 218.0, 32.0
	default:
		return 18.6, 218.0, 32.0 // fallback to Pro
	}
}

func resolveBudgets(tier string) ResolvedBudgets {
	def5h, defWeekly, defSonnet := tierDefaults(tier)
	cfg := readAppConfig()

	b := ResolvedBudgets{
		FiveHour:     def5h,
		Weekly:       defWeekly,
		WeeklySonnet: defSonnet,
	}
	if cfg.BudgetOverrides != nil {
		if cfg.BudgetOverrides.FiveHour != nil {
			b.FiveHour = *cfg.BudgetOverrides.FiveHour
		}
		if cfg.BudgetOverrides.Weekly != nil {
			b.Weekly = *cfg.BudgetOverrides.Weekly
		}
		if cfg.BudgetOverrides.WeeklySonnet != nil {
			b.WeeklySonnet = *cfg.BudgetOverrides.WeeklySonnet
		}
	}
	return b
}
