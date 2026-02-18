package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// --- JSONL deserialization structs ---

type JournalEntry struct {
	EntryType string   `json:"type"`
	Timestamp *string  `json:"timestamp"`
	Message   *Message `json:"message"`
}

type Message struct {
	Model *string `json:"model"`
	Usage *Usage  `json:"usage"`
}

type Usage struct {
	InputTokens          *uint64 `json:"input_tokens"`
	OutputTokens         *uint64 `json:"output_tokens"`
	CacheReadInputTokens *uint64 `json:"cache_read_input_tokens"`
	CacheCreationTokens  *uint64 `json:"cache_creation_input_tokens"`
}

// --- Output structs (sent to frontend as JSON) ---

type TokenCounts struct {
	InputTokens     uint64 `json:"input_tokens"`
	OutputTokens    uint64 `json:"output_tokens"`
	CacheReadTokens uint64 `json:"cache_read_tokens"`
	CacheCreateTokens uint64 `json:"cache_create_tokens"`
}

type ModelUsageOut struct {
	Tokens  TokenCounts `json:"tokens"`
	CostUSD float64     `json:"cost_usd"`
}

type UsageWindow struct {
	TotalCostUSD      float64                  `json:"total_cost_usd"`
	InputTokens       uint64                   `json:"input_tokens"`
	OutputTokens      uint64                   `json:"output_tokens"`
	CacheReadTokens   uint64                   `json:"cache_read_tokens"`
	CacheCreateTokens uint64                   `json:"cache_create_tokens"`
	ByModel           map[string]ModelUsageOut  `json:"by_model"`
	OldestEntryTS     *string                  `json:"oldest_entry_ts"`
	NewestEntryTS     *string                  `json:"newest_entry_ts"`
	EntryCount        uint64                   `json:"entry_count"`
}

type WindowRateInfo struct {
	BudgetUSD float64     `json:"budget_usd"`
	CostUSD   float64     `json:"cost_usd"`
	Percent   float64     `json:"percent"`
	ResetTS   *string     `json:"reset_ts"`
	Window    UsageWindow `json:"window"`
}

type RateLimitInfo struct {
	TierName        string         `json:"tier_name"`
	FiveHour        WindowRateInfo `json:"five_hour"`
	Weekly          WindowRateInfo `json:"weekly"`
	WeeklySonnet    WindowRateInfo `json:"weekly_sonnet"`
	ApiAvailable    bool           `json:"api_available"`
	RateLimitStatus *string        `json:"rate_limit_status"`
}

// --- Internal entry ---

type usageEntry struct {
	model             string
	timestamp         time.Time
	inputTokens       uint64
	outputTokens      uint64
	cacheReadTokens   uint64
	cacheCreateTokens uint64
}

// --- UsageService (bound to Wails for JS calls) ---

type UsageService struct{}

func (s *UsageService) GetUsage() (*RateLimitInfo, error) {
	tier := readTier()
	budgets := resolveBudgets(tier)
	token := readOAuthToken()

	var apiData *ApiUsageResponse
	if token != "" {
		apiData, _ = fetchUsage(token)
	}

	return getUsageSummaryWithAPI(&budgets, tier, apiData)
}

// --- Functions ---

func getClaudeProjectsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

func scanJSONLFiles(projectsDir string, since time.Time) []string {
	var files []string

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subDir := filepath.Join(projectsDir, entry.Name())
		subEntries, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}
		for _, f := range subEntries {
			if f.IsDir() {
				continue
			}
			if !strings.HasSuffix(f.Name(), ".jsonl") {
				continue
			}
			fPath := filepath.Join(subDir, f.Name())
			info, err := f.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(since) {
				continue
			}
			files = append(files, fPath)
		}
	}
	return files
}

func parseJSONLFile(path string, since time.Time) []usageEntry {
	var entries []usageEntry

	file, err := os.Open(path)
	if err != nil {
		return entries
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry JournalEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.EntryType != "assistant" {
			continue
		}
		if entry.Message == nil || entry.Message.Usage == nil || entry.Timestamp == nil {
			continue
		}

		ts, err := time.Parse(time.RFC3339Nano, *entry.Timestamp)
		if err != nil {
			// Try other formats
			ts, err = time.Parse(time.RFC3339, *entry.Timestamp)
			if err != nil {
				continue
			}
		}

		if ts.Before(since) {
			continue
		}

		model := "unknown"
		if entry.Message.Model != nil {
			model = *entry.Message.Model
		}

		u := entry.Message.Usage
		entries = append(entries, usageEntry{
			model:             model,
			timestamp:         ts,
			inputTokens:       derefU64(u.InputTokens),
			outputTokens:      derefU64(u.OutputTokens),
			cacheReadTokens:   derefU64(u.CacheReadInputTokens),
			cacheCreateTokens: derefU64(u.CacheCreationTokens),
		})
	}
	return entries
}

func derefU64(p *uint64) uint64 {
	if p == nil {
		return 0
	}
	return *p
}

func aggregateEntries(entries []usageEntry, since time.Time) UsageWindow {
	byModel := make(map[string]ModelUsageOut)
	var totalInput, totalOutput, totalCacheRead, totalCacheCreate, count uint64
	var totalCost float64
	var oldest, newest *time.Time

	for i := range entries {
		e := &entries[i]
		if e.timestamp.Before(since) {
			continue
		}
		count++

		if oldest == nil || e.timestamp.Before(*oldest) {
			t := e.timestamp
			oldest = &t
		}
		if newest == nil || e.timestamp.After(*newest) {
			t := e.timestamp
			newest = &t
		}

		totalInput += e.inputTokens
		totalOutput += e.outputTokens
		totalCacheRead += e.cacheReadTokens
		totalCacheCreate += e.cacheCreateTokens

		cost := calculateCost(e.model, e.inputTokens, e.outputTokens, e.cacheReadTokens, e.cacheCreateTokens)
		totalCost += cost

		mu := byModel[e.model]
		mu.Tokens.InputTokens += e.inputTokens
		mu.Tokens.OutputTokens += e.outputTokens
		mu.Tokens.CacheReadTokens += e.cacheReadTokens
		mu.Tokens.CacheCreateTokens += e.cacheCreateTokens
		mu.CostUSD += cost
		byModel[e.model] = mu
	}

	var oldestStr, newestStr *string
	if oldest != nil {
		s := oldest.UTC().Format(time.RFC3339)
		oldestStr = &s
	}
	if newest != nil {
		s := newest.UTC().Format(time.RFC3339)
		newestStr = &s
	}

	return UsageWindow{
		TotalCostUSD:      math.Round(totalCost*1000) / 1000,
		InputTokens:       totalInput,
		OutputTokens:      totalOutput,
		CacheReadTokens:   totalCacheRead,
		CacheCreateTokens: totalCacheCreate,
		ByModel:           byModel,
		OldestEntryTS:     oldestStr,
		NewestEntryTS:     newestStr,
		EntryCount:        count,
	}
}

func computeResetTS(oldestTS *string, windowDuration time.Duration) *string {
	if oldestTS == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *oldestTS)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, *oldestTS)
		if err != nil {
			return nil
		}
	}
	reset := t.Add(windowDuration).UTC().Format(time.RFC3339)
	return &reset
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func getUsageSummary(budgets *ResolvedBudgets, tier string) (*RateLimitInfo, error) {
	projectsDir := getClaudeProjectsDir()
	if projectsDir == "" {
		return nil, fmt.Errorf("could not find Claude projects directory")
	}
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Claude projects directory does not exist")
	}

	now := time.Now().UTC()
	weekAgo := now.Add(-7 * 24 * time.Hour)
	fiveHoursAgo := now.Add(-5 * time.Hour)

	files := scanJSONLFiles(projectsDir, weekAgo)

	var allEntries []usageEntry
	for _, f := range files {
		allEntries = append(allEntries, parseJSONLFile(f, weekAgo)...)
	}

	fiveHour := aggregateEntries(allEntries, fiveHoursAgo)
	weekly := aggregateEntries(allEntries, weekAgo)

	// Weekly Sonnet-only
	var sonnetEntries []usageEntry
	for _, e := range allEntries {
		if strings.Contains(strings.ToLower(e.model), "sonnet") {
			sonnetEntries = append(sonnetEntries, e)
		}
	}
	weeklySonnet := aggregateEntries(sonnetEntries, weekAgo)

	fiveHourPct := clamp(fiveHour.TotalCostUSD/budgets.FiveHour*100, 0, 100)
	weeklyPct := clamp(weekly.TotalCostUSD/budgets.Weekly*100, 0, 100)
	weeklySonnetPct := clamp(weeklySonnet.TotalCostUSD/budgets.WeeklySonnet*100, 0, 100)

	fiveHourReset := computeResetTS(fiveHour.OldestEntryTS, 5*time.Hour)
	weeklyReset := computeResetTS(weekly.OldestEntryTS, 7*24*time.Hour)
	weeklySonnetReset := computeResetTS(weeklySonnet.OldestEntryTS, 7*24*time.Hour)

	return &RateLimitInfo{
		TierName: tier,
		FiveHour: WindowRateInfo{
			BudgetUSD: budgets.FiveHour,
			CostUSD:   fiveHour.TotalCostUSD,
			Percent:   math.Round(fiveHourPct*10) / 10,
			ResetTS:   fiveHourReset,
			Window:    fiveHour,
		},
		Weekly: WindowRateInfo{
			BudgetUSD: budgets.Weekly,
			CostUSD:   weekly.TotalCostUSD,
			Percent:   math.Round(weeklyPct*10) / 10,
			ResetTS:   weeklyReset,
			Window:    weekly,
		},
		WeeklySonnet: WindowRateInfo{
			BudgetUSD: budgets.WeeklySonnet,
			CostUSD:   weeklySonnet.TotalCostUSD,
			Percent:   math.Round(weeklySonnetPct*10) / 10,
			ResetTS:   weeklySonnetReset,
			Window:    weeklySonnet,
		},
		ApiAvailable:    false,
		RateLimitStatus: nil,
	}, nil
}

func getUsageSummaryWithAPI(budgets *ResolvedBudgets, tier string, apiUsage *ApiUsageResponse) (*RateLimitInfo, error) {
	info, err := getUsageSummary(budgets, tier)
	if err != nil {
		return nil, err
	}

	if apiUsage == nil {
		return info, nil
	}

	info.ApiAvailable = true

	if apiUsage.FiveHour != nil {
		if apiUsage.FiveHour.Utilization != nil {
			info.FiveHour.Percent = *apiUsage.FiveHour.Utilization
		}
		if apiUsage.FiveHour.ResetsAt != nil {
			info.FiveHour.ResetTS = apiUsage.FiveHour.ResetsAt
		}
	}

	if apiUsage.SevenDay != nil {
		if apiUsage.SevenDay.Utilization != nil {
			info.Weekly.Percent = *apiUsage.SevenDay.Utilization
		}
		if apiUsage.SevenDay.ResetsAt != nil {
			info.Weekly.ResetTS = apiUsage.SevenDay.ResetsAt
		}
	}

	if apiUsage.SevenDaySonnet != nil {
		if apiUsage.SevenDaySonnet.Utilization != nil {
			info.WeeklySonnet.Percent = *apiUsage.SevenDaySonnet.Utilization
		}
		if apiUsage.SevenDaySonnet.ResetsAt != nil {
			info.WeeklySonnet.ResetTS = apiUsage.SevenDaySonnet.ResetsAt
		}
	}

	return info, nil
}
