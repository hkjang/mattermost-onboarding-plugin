package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const defaultTemplatesJSON = `[
  {
    "template_id": "common-ko",
    "template_type": "common",
    "lang": "ko",
    "title": "Mattermost 온보딩 안내",
    "body": "{{user_name}}님, Mattermost에 오신 것을 환영합니다.\n\n아래 필수 안내를 먼저 확인해 주세요.",
    "active_yn": true,
    "version": 1
  },
  {
    "template_id": "dept-default-ko",
    "template_type": "department",
    "dept_code": "DEFAULT",
    "lang": "ko",
    "title": "부서별 안내",
    "body": "부서 전용 위키와 운영 가이드는 아래 링크를 확인해 주세요.",
    "active_yn": true,
    "version": 1
  }
]`

const defaultLinksJSON = `[
  {
    "link_id": "common-security",
    "template_id": "common-ko",
    "link_title": "필수 공지 및 보안 가이드",
    "link_desc": "신규 사용자가 우선 확인해야 하는 공통 안내입니다.",
    "link_url": "https://confluence.example.com/display/COMMON/SECURITY",
    "sort_order": 10,
    "active_yn": true
  },
  {
    "link_id": "common-collab",
    "template_id": "common-ko",
    "link_title": "협업 채널 이용 가이드",
    "link_desc": "채널 구조와 협업 규칙을 확인합니다.",
    "link_url": "https://confluence.example.com/display/COMMON/COLLABORATION",
    "sort_order": 20,
    "active_yn": true
  },
  {
    "link_id": "dept-default",
    "template_id": "dept-default-ko",
    "link_title": "부서별 문서 모음",
    "link_desc": "소속 부서용 문서와 체크리스트입니다.",
    "link_url": "https://confluence.example.com/display/DEPT/DEFAULT",
    "sort_order": 10,
    "active_yn": true
  }
]`

const defaultExclusionsJSON = `[
  {
    "exclusion_id": "exclude-bots",
    "rule_type": "is_bot",
    "rule_value": "true",
    "active_yn": true
  },
  {
    "exclusion_id": "exclude-deleted",
    "rule_type": "deleted",
    "rule_value": "true",
    "active_yn": true
  }
]`

func (p *Plugin) getRuntimeConfiguration() *runtimeConfiguration {
	config := p.getConfiguration()
	if config.runtime != nil {
		return config.runtime
	}

	runtime, err := buildRuntimeConfiguration(config)
	if err != nil {
		p.API.LogError("Failed to build runtime configuration; using safe defaults", "error", err.Error())
		return &runtimeConfiguration{
			Enabled:                false,
			SenderBotUsername:      "onboarding.bot",
			SenderBotDisplayName:   "Onboarding Bot",
			DefaultLanguage:        "ko",
			FallbackDepartmentCode: "DEFAULT",
			RetryIntervalMinutes:   5,
			RetryMaxAttempts:       3,
		}
	}

	return runtime
}

func buildRuntimeConfiguration(config *configuration) (*runtimeConfiguration, error) {
	runtime := &runtimeConfiguration{
		Enabled:                config.EnableAutoSend,
		SenderBotUsername:      strings.TrimSpace(config.SenderBotUsername),
		SenderBotDisplayName:   strings.TrimSpace(config.SenderBotDisplayName),
		DefaultLanguage:        normalizeLanguage(config.DefaultLanguage),
		FallbackDepartmentCode: strings.TrimSpace(config.FallbackDepartmentCode),
		InitialDelaySeconds:    config.InitialDelaySeconds,
		RetryIntervalMinutes:   config.RetryIntervalMinutes,
		RetryMaxAttempts:       config.RetryMaxAttempts,
	}

	if runtime.SenderBotUsername == "" {
		runtime.SenderBotUsername = "onboarding.bot"
	}
	if runtime.SenderBotDisplayName == "" {
		runtime.SenderBotDisplayName = "Onboarding Bot"
	}
	if runtime.DefaultLanguage == "" {
		runtime.DefaultLanguage = "ko"
	}
	if runtime.FallbackDepartmentCode == "" {
		runtime.FallbackDepartmentCode = "DEFAULT"
	}
	if runtime.RetryIntervalMinutes <= 0 {
		runtime.RetryIntervalMinutes = 5
	}
	if runtime.RetryMaxAttempts <= 0 {
		runtime.RetryMaxAttempts = 3
	}
	if runtime.InitialDelaySeconds < 0 {
		runtime.InitialDelaySeconds = 0
	}

	if err := decodeJSONArray("TemplatesJSON", config.TemplatesJSON, defaultTemplatesJSON, &runtime.Templates); err != nil {
		return nil, err
	}
	if err := decodeJSONArray("LinksJSON", config.LinksJSON, defaultLinksJSON, &runtime.Links); err != nil {
		return nil, err
	}
	if err := decodeJSONArray("DepartmentMappingsJSON", config.DepartmentMappingsJSON, "[]", &runtime.Mappings); err != nil {
		return nil, err
	}
	if err := decodeJSONArray("ExclusionRulesJSON", config.ExclusionRulesJSON, defaultExclusionsJSON, &runtime.Exclusions); err != nil {
		return nil, err
	}

	normalizeTemplates(runtime.Templates, runtime.DefaultLanguage)
	normalizeLinks(runtime.Links)
	normalizeMappings(runtime.Mappings)
	normalizeExclusions(runtime.Exclusions)

	if err := validateRuntimeConfiguration(runtime); err != nil {
		return nil, err
	}

	return runtime, nil
}

func decodeJSONArray(fieldName, rawValue, fallback string, target interface{}) error {
	candidate := strings.TrimSpace(rawValue)
	if candidate == "" {
		candidate = fallback
	}

	if err := json.Unmarshal([]byte(candidate), target); err != nil {
		return fmt.Errorf("%s must contain valid JSON: %w", fieldName, err)
	}

	return nil
}

func normalizeTemplates(templates []onboardingTemplate, defaultLanguage string) {
	for i := range templates {
		templates[i].ID = strings.TrimSpace(templates[i].ID)
		templates[i].Type = strings.TrimSpace(templates[i].Type)
		templates[i].DepartmentCode = strings.TrimSpace(templates[i].DepartmentCode)
		templates[i].Lang = normalizeLanguage(templates[i].Lang)
		templates[i].Title = strings.TrimSpace(templates[i].Title)
		templates[i].Body = strings.TrimSpace(templates[i].Body)
		if templates[i].Lang == "" {
			templates[i].Lang = defaultLanguage
		}
		if templates[i].Version == 0 {
			templates[i].Version = 1
		}
	}

	sort.SliceStable(templates, func(i, j int) bool {
		return templates[i].ID < templates[j].ID
	})
}

func normalizeLinks(links []onboardingLink) {
	sort.SliceStable(links, func(i, j int) bool {
		if links[i].TemplateID == links[j].TemplateID {
			if links[i].SortOrder == links[j].SortOrder {
				return links[i].ID < links[j].ID
			}
			return links[i].SortOrder < links[j].SortOrder
		}
		return links[i].TemplateID < links[j].TemplateID
	})

	for i := range links {
		links[i].ID = strings.TrimSpace(links[i].ID)
		links[i].TemplateID = strings.TrimSpace(links[i].TemplateID)
		links[i].Title = strings.TrimSpace(links[i].Title)
		links[i].Description = strings.TrimSpace(links[i].Description)
		links[i].URL = strings.TrimSpace(links[i].URL)
	}
}

func normalizeMappings(mappings []departmentMapping) {
	for i := range mappings {
		mappings[i].SourceDepartmentCode = strings.TrimSpace(mappings[i].SourceDepartmentCode)
		mappings[i].SourceDepartmentName = strings.TrimSpace(mappings[i].SourceDepartmentName)
		mappings[i].TemplateDepartmentCode = strings.TrimSpace(mappings[i].TemplateDepartmentCode)
		mappings[i].DepartmentName = strings.TrimSpace(mappings[i].DepartmentName)
		mappings[i].OrganizationName = strings.TrimSpace(mappings[i].OrganizationName)
	}
}

func normalizeExclusions(exclusions []onboardingExclusionRule) {
	for i := range exclusions {
		exclusions[i].ID = strings.TrimSpace(exclusions[i].ID)
		exclusions[i].RuleType = strings.TrimSpace(exclusions[i].RuleType)
		exclusions[i].RuleValue = strings.TrimSpace(exclusions[i].RuleValue)
	}
}

func validateRuntimeConfiguration(config *runtimeConfiguration) error {
	templateIDs := make(map[string]struct{}, len(config.Templates))
	hasActiveCommonTemplate := false

	for _, template := range config.Templates {
		if template.ID == "" {
			return fmt.Errorf("TemplatesJSON contains a template without template_id")
		}
		if template.Type != templateTypeCommon && template.Type != templateTypeDepartment {
			return fmt.Errorf("template %s has unsupported template_type %q", template.ID, template.Type)
		}
		if template.Type == templateTypeDepartment && template.DepartmentCode == "" {
			return fmt.Errorf("department template %s must define dept_code", template.ID)
		}
		if _, exists := templateIDs[template.ID]; exists {
			return fmt.Errorf("template_id %s is duplicated", template.ID)
		}
		templateIDs[template.ID] = struct{}{}
		if template.Active && template.Type == templateTypeCommon {
			hasActiveCommonTemplate = true
		}
	}

	if !hasActiveCommonTemplate {
		return fmt.Errorf("at least one active common template is required")
	}

	for _, link := range config.Links {
		if link.ID == "" {
			return fmt.Errorf("LinksJSON contains a link without link_id")
		}
		if link.TemplateID == "" {
			return fmt.Errorf("link %s must define template_id", link.ID)
		}
		if _, exists := templateIDs[link.TemplateID]; !exists {
			return fmt.Errorf("link %s references unknown template %s", link.ID, link.TemplateID)
		}
	}

	return nil
}

func normalizeLanguage(language string) string {
	return strings.ToLower(strings.TrimSpace(language))
}
