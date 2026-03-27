package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type onboardingAdminConfig struct {
	EnableAutoSend         bool   `json:"enable_auto_send"`
	SenderBotUsername      string `json:"sender_bot_username"`
	SenderBotDisplayName   string `json:"sender_bot_display_name"`
	DefaultLanguage        string `json:"default_language"`
	FallbackDepartmentCode string `json:"fallback_department_code"`
	InitialDelaySeconds    int    `json:"initial_delay_seconds"`
	RetryIntervalMinutes   int    `json:"retry_interval_minutes"`
	RetryMaxAttempts       int    `json:"retry_max_attempts"`
	TemplatesJSON          string `json:"templates_json"`
	LinksJSON              string `json:"links_json"`
	DepartmentMappingsJSON string `json:"department_mappings_json"`
	ExclusionRulesJSON     string `json:"exclusion_rules_json"`
}

func (p *Plugin) getAdminConfiguration() onboardingAdminConfig {
	config := p.getConfiguration()
	runtime := p.getRuntimeConfiguration()

	return onboardingAdminConfig{
		EnableAutoSend:         runtime.Enabled,
		SenderBotUsername:      runtime.SenderBotUsername,
		SenderBotDisplayName:   runtime.SenderBotDisplayName,
		DefaultLanguage:        runtime.DefaultLanguage,
		FallbackDepartmentCode: runtime.FallbackDepartmentCode,
		InitialDelaySeconds:    runtime.InitialDelaySeconds,
		RetryIntervalMinutes:   runtime.RetryIntervalMinutes,
		RetryMaxAttempts:       runtime.RetryMaxAttempts,
		TemplatesJSON:          displayJSONSetting(config.TemplatesJSON, defaultTemplatesJSON),
		LinksJSON:              displayJSONSetting(config.LinksJSON, defaultLinksJSON),
		DepartmentMappingsJSON: displayJSONSetting(config.DepartmentMappingsJSON, "[]"),
		ExclusionRulesJSON:     displayJSONSetting(config.ExclusionRulesJSON, defaultExclusionsJSON),
	}
}

func (p *Plugin) saveAdminConfiguration(input onboardingAdminConfig) (onboardingAdminConfig, error) {
	next := p.getConfiguration().Clone()
	next.EnableAutoSend = input.EnableAutoSend
	next.SenderBotUsername = strings.TrimSpace(input.SenderBotUsername)
	next.SenderBotDisplayName = strings.TrimSpace(input.SenderBotDisplayName)
	next.DefaultLanguage = strings.TrimSpace(input.DefaultLanguage)
	next.FallbackDepartmentCode = strings.TrimSpace(input.FallbackDepartmentCode)
	next.InitialDelaySeconds = input.InitialDelaySeconds
	next.RetryIntervalMinutes = input.RetryIntervalMinutes
	next.RetryMaxAttempts = input.RetryMaxAttempts

	var err error
	if next.TemplatesJSON, err = normalizeJSONSetting(input.TemplatesJSON); err != nil {
		return onboardingAdminConfig{}, fmt.Errorf("Templates JSON must be valid JSON: %w", err)
	}
	if next.LinksJSON, err = normalizeJSONSetting(input.LinksJSON); err != nil {
		return onboardingAdminConfig{}, fmt.Errorf("Links JSON must be valid JSON: %w", err)
	}
	if next.DepartmentMappingsJSON, err = normalizeJSONSetting(input.DepartmentMappingsJSON); err != nil {
		return onboardingAdminConfig{}, fmt.Errorf("Department mappings JSON must be valid JSON: %w", err)
	}
	if next.ExclusionRulesJSON, err = normalizeJSONSetting(input.ExclusionRulesJSON); err != nil {
		return onboardingAdminConfig{}, fmt.Errorf("Exclusion rules JSON must be valid JSON: %w", err)
	}

	if _, err := buildRuntimeConfiguration(next); err != nil {
		return onboardingAdminConfig{}, err
	}

	pluginConfig := p.API.GetPluginConfig()
	if pluginConfig == nil {
		pluginConfig = map[string]interface{}{}
	}

	pluginConfig["EnableAutoSend"] = next.EnableAutoSend
	pluginConfig["SenderBotUsername"] = next.SenderBotUsername
	pluginConfig["SenderBotDisplayName"] = next.SenderBotDisplayName
	pluginConfig["DefaultLanguage"] = next.DefaultLanguage
	pluginConfig["FallbackDepartmentCode"] = next.FallbackDepartmentCode
	pluginConfig["InitialDelaySeconds"] = next.InitialDelaySeconds
	pluginConfig["RetryIntervalMinutes"] = next.RetryIntervalMinutes
	pluginConfig["RetryMaxAttempts"] = next.RetryMaxAttempts
	pluginConfig["TemplatesJSON"] = next.TemplatesJSON
	pluginConfig["LinksJSON"] = next.LinksJSON
	pluginConfig["DepartmentMappingsJSON"] = next.DepartmentMappingsJSON
	pluginConfig["ExclusionRulesJSON"] = next.ExclusionRulesJSON

	if appErr := p.API.SavePluginConfig(pluginConfig); appErr != nil {
		return onboardingAdminConfig{}, fmt.Errorf("failed to save plugin configuration: %w", appErr)
	}

	if err := p.OnConfigurationChange(); err != nil {
		return onboardingAdminConfig{}, err
	}

	return p.getAdminConfiguration(), nil
}

func normalizeJSONSetting(raw string) (string, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return "", nil
	}

	var payload interface{}
	if err := json.Unmarshal([]byte(candidate), &payload); err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

func displayJSONSetting(raw string, fallback string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		candidate = strings.TrimSpace(fallback)
	}
	if candidate == "" {
		return ""
	}

	formatted, err := normalizeJSONSetting(candidate)
	if err != nil {
		return candidate
	}

	return formatted
}
