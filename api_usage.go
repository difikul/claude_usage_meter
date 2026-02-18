package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WindowUsage struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    *string  `json:"resets_at"`
}

type ExtraUsage struct {
	IsEnabled    *bool    `json:"is_enabled"`
	MonthlyLimit *float64 `json:"monthly_limit"`
	UsedCredits  *float64 `json:"used_credits"`
	Utilization  *float64 `json:"utilization"`
}

type ApiUsageResponse struct {
	FiveHour       *WindowUsage `json:"five_hour"`
	SevenDay       *WindowUsage `json:"seven_day"`
	SevenDaySonnet *WindowUsage `json:"seven_day_sonnet"`
	ExtraUsage     *ExtraUsage  `json:"extra_usage"`
}

func fetchUsage(accessToken string) (*ApiUsageResponse, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result ApiUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
