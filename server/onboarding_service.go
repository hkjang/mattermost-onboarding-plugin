package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const automaticRequester = "system:auto"

var errNoOnboardingMessage = errors.New("no onboarding message is available")

func (p *Plugin) UserHasBeenCreated(_ *plugin.Context, user *model.User) {
	if user == nil || user.Id == "" {
		return
	}

	config := p.getRuntimeConfiguration()
	if !config.Enabled {
		return
	}

	if config.InitialDelaySeconds > 0 {
		if err := p.enqueueOnboardingRetry(user.Id, automaticRequester, "", time.Duration(config.InitialDelaySeconds)*time.Second); err != nil {
			p.API.LogError("Failed to enqueue delayed onboarding message", "user_id", user.Id, "error", err.Error())
		}
		return
	}

	go func(userID string) {
		if _, err := p.sendOnboardingByUserID(userID, sendModeAutomatic, automaticRequester); err != nil {
			p.API.LogError("Automatic onboarding delivery failed", "user_id", userID, "error", err.Error())
		}
	}(user.Id)
}

func (p *Plugin) sendOnboardingByUserID(userID, mode, requestedBy string) (*onboardingSendLog, error) {
	if mode == sendModeAutomatic {
		claimedState, claimed, err := p.claimAutomaticSend(userID)
		if err != nil {
			return nil, err
		}
		if !claimed {
			existingState, stateErr := p.getAutomaticSendState(userID)
			if stateErr != nil {
				return nil, stateErr
			}
			if existingState == nil {
				return nil, nil
			}

			latestLog, logErr := p.getLatestSendLog(userID)
			if logErr != nil {
				return nil, logErr
			}

			switch existingState.Status {
			case sendStatusSent, sendStatusSkipped, sendStatusFailed:
				return latestLog, nil
			default:
				return nil, nil
			}
		}

		user, userErr := p.client.User.Get(userID)
		if userErr != nil {
			_ = p.releaseAutomaticSend(userID)
			if retryErr := p.enqueueOnboardingRetry(userID, requestedBy, userErr.Error(), p.retryInterval()); retryErr != nil {
				p.API.LogError("Failed to enqueue onboarding retry after user lookup error", "user_id", userID, "error", retryErr.Error())
			}
			return nil, fmt.Errorf("failed to load onboarding target user: %w", userErr)
		}

		log, retryable, deliverErr := p.deliverOnboarding(user, mode, requestedBy)
		if deliverErr != nil {
			if retryable {
				_ = p.releaseAutomaticSend(userID)
				if retryErr := p.enqueueOnboardingRetry(userID, requestedBy, deliverErr.Error(), p.retryInterval()); retryErr != nil {
					p.API.LogError("Failed to enqueue onboarding retry", "user_id", userID, "error", retryErr.Error())
				}
			} else {
				claimedState.Status = sendStatusFailed
				claimedState.UpdatedAt = time.Now().UTC().UnixMilli()
				if log != nil {
					claimedState.LogID = log.ID
				}
				_ = p.finalizeAutomaticSend(userID, claimedState)
			}
			return log, deliverErr
		}

		if log != nil {
			claimedState.Status = log.Status
			claimedState.LogID = log.ID
			claimedState.UpdatedAt = time.Now().UTC().UnixMilli()
			if err := p.finalizeAutomaticSend(userID, claimedState); err != nil {
				return log, err
			}
		}

		return log, nil
	}

	user, err := p.client.User.Get(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load onboarding target user: %w", err)
	}

	log, _, deliverErr := p.deliverOnboarding(user, mode, requestedBy)
	return log, deliverErr
}

func (p *Plugin) deliverOnboarding(user *model.User, mode, requestedBy string) (*onboardingSendLog, bool, error) {
	config := p.getRuntimeConfiguration()
	nowMillis := time.Now().UTC().UnixMilli()

	if exclusionReason, err := p.shouldExcludeUser(user, config); err != nil {
		log := p.newSendLog(user, mode, requestedBy, sendStatusFailed, nil, err.Error(), nowMillis)
		_ = p.saveSendLog(log)
		return log, true, fmt.Errorf("failed to evaluate exclusion rules: %w", err)
	} else if exclusionReason != "" {
		log := p.newSendLog(user, mode, requestedBy, sendStatusSkipped, nil, exclusionReason, nowMillis)
		_ = p.saveSendLog(log)
		return log, false, nil
	}

	rendered, err := p.renderOnboardingMessage(user, nil)
	if err != nil {
		log := p.newSendLog(user, mode, requestedBy, sendStatusSkipped, nil, err.Error(), nowMillis)
		if !errors.Is(err, errNoOnboardingMessage) {
			log.Status = sendStatusFailed
		}
		_ = p.saveSendLog(log)
		if errors.Is(err, errNoOnboardingMessage) {
			return log, false, nil
		}
		return log, true, err
	}

	botUserID := p.getBotUserID()
	if botUserID == "" {
		if ensureErr := p.ensureSenderBot(); ensureErr != nil {
			log := p.newSendLog(user, mode, requestedBy, sendStatusFailed, rendered.AppliedTemplateIDs, ensureErr.Error(), nowMillis)
			_ = p.saveSendLog(log)
			return log, true, ensureErr
		}
		botUserID = p.getBotUserID()
	}

	post := &model.Post{Message: rendered.Message}
	if err := p.client.Post.DM(botUserID, user.Id, post); err != nil {
		log := p.newSendLog(user, mode, requestedBy, sendStatusFailed, rendered.AppliedTemplateIDs, err.Error(), nowMillis)
		_ = p.saveSendLog(log)
		return log, true, fmt.Errorf("failed to send onboarding DM: %w", err)
	}

	log := p.newSendLog(user, mode, requestedBy, sendStatusSent, rendered.AppliedTemplateIDs, "", nowMillis)
	log.DepartmentCode = rendered.DepartmentCode
	if err := p.saveSendLog(log); err != nil {
		return log, false, err
	}

	return log, false, nil
}

func (p *Plugin) processDueQueueItems() {
	items, err := p.listDueQueueItems(100, time.Now().UTC())
	if err != nil {
		p.API.LogError("Failed to list queued onboarding retries", "error", err.Error())
		return
	}

	for _, item := range items {
		p.retryQueuedOnboarding(item)
	}
}

func (p *Plugin) retryQueuedOnboarding(item onboardingQueueItem) {
	config := p.getRuntimeConfiguration()
	if item.AttemptCount >= config.RetryMaxAttempts {
		item.Status = queueStatusFailed
		if err := p.saveQueueItem(&item); err != nil {
			p.API.LogError("Failed to update exhausted onboarding queue item", "user_id", item.TargetUserID, "error", err.Error())
		}
		return
	}

	item.AttemptCount++
	item.NextAttemptAt = time.Now().UTC().Add(p.retryInterval()).UnixMilli()
	if err := p.saveQueueItem(&item); err != nil {
		p.API.LogError("Failed to persist onboarding retry attempt", "user_id", item.TargetUserID, "error", err.Error())
	}

	log, err := p.sendOnboardingByUserID(item.TargetUserID, sendModeAutomatic, item.RequestedBy)
	if err != nil {
		item.LastError = err.Error()
		item.Status = queueStatusPending
		if saveErr := p.saveQueueItem(&item); saveErr != nil {
			p.API.LogError("Failed to persist onboarding retry failure", "user_id", item.TargetUserID, "error", saveErr.Error())
		}
		return
	}

	if log != nil && (log.Status == sendStatusSent || log.Status == sendStatusSkipped) {
		if deleteErr := p.deleteQueueItem(item.TargetUserID); deleteErr != nil {
			p.API.LogError("Failed to clear onboarding queue item", "user_id", item.TargetUserID, "error", deleteErr.Error())
		}
		return
	}

	if log != nil && log.Status == sendStatusFailed {
		item.Status = queueStatusFailed
		item.LastError = log.ErrorMessage
		if saveErr := p.saveQueueItem(&item); saveErr != nil {
			p.API.LogError("Failed to persist terminal onboarding queue failure", "user_id", item.TargetUserID, "error", saveErr.Error())
		}
	}
}

func (p *Plugin) enqueueOnboardingRetry(userID, requestedBy, lastError string, delay time.Duration) error {
	now := time.Now().UTC()
	item, err := p.getQueueItem(userID)
	if err != nil {
		return err
	}

	if item == nil {
		item = &onboardingQueueItem{
			ID:           fmt.Sprintf("queue-%s", userID),
			TargetUserID: userID,
			RequestedBy:  requestedBy,
			RequestedAt:  now.UnixMilli(),
			Status:       queueStatusPending,
		}
	}

	item.Status = queueStatusPending
	item.LastError = lastError
	if requestedBy != "" {
		item.RequestedBy = requestedBy
	}
	item.NextAttemptAt = now.Add(delay).UnixMilli()

	return p.saveQueueItem(item)
}

func (p *Plugin) retryInterval() time.Duration {
	config := p.getRuntimeConfiguration()
	return time.Duration(config.RetryIntervalMinutes) * time.Minute
}

func (p *Plugin) newSendLog(user *model.User, mode, requestedBy, status string, templateIDs []string, errorMessage string, sentAt int64) *onboardingSendLog {
	log := &onboardingSendLog{
		ID:           fmt.Sprintf("%d-%s-%s", sentAt, mode, user.Id),
		UserID:       user.Id,
		Username:     user.Username,
		TemplateIDs:  templateIDs,
		SentAt:       sentAt,
		Status:       status,
		ErrorMessage: strings.TrimSpace(errorMessage),
		Mode:         mode,
		RequestedBy:  requestedBy,
	}

	profile := p.resolveOnboardingProfile(user, nil)
	log.DepartmentCode = profile.DepartmentCode

	return log
}

func (p *Plugin) shouldExcludeUser(user *model.User, config *runtimeConfiguration) (string, error) {
	requiresGroupLookup := false
	for _, rule := range config.Exclusions {
		if !rule.Active {
			continue
		}
		if rule.RuleType == exclusionRuleGroupID || rule.RuleType == exclusionRuleGroupName {
			requiresGroupLookup = true
			break
		}
	}

	groupIDs := map[string]struct{}{}
	groupNames := map[string]struct{}{}
	if requiresGroupLookup {
		groups, err := p.client.Group.ListForUser(user.Id)
		if err != nil {
			p.API.LogWarn("Failed to evaluate group-based exclusion rules; continuing without group filters", "user_id", user.Id, "error", err.Error())
		} else {
			for _, group := range groups {
				groupIDs[strings.ToLower(group.Id)] = struct{}{}
				if group.Name != nil {
					name := strings.ToLower(strings.TrimSpace(*group.Name))
					if name != "" {
						groupNames[name] = struct{}{}
					}
				}
			}
		}
	}

	emailDomain := ""
	if parts := strings.Split(strings.ToLower(strings.TrimSpace(user.Email)), "@"); len(parts) == 2 {
		emailDomain = parts[1]
	}

	for _, rule := range config.Exclusions {
		if !rule.Active {
			continue
		}

		value := strings.ToLower(strings.TrimSpace(rule.RuleValue))
		switch rule.RuleType {
		case exclusionRuleEmailDomain:
			if value != "" && emailDomain == value {
				return fmt.Sprintf("excluded by email domain rule %s", rule.ID), nil
			}
		case exclusionRuleUsername:
			if value != "" && strings.EqualFold(user.Username, value) {
				return fmt.Sprintf("excluded by username rule %s", rule.ID), nil
			}
		case exclusionRuleUserID:
			if value != "" && strings.EqualFold(user.Id, value) {
				return fmt.Sprintf("excluded by user_id rule %s", rule.ID), nil
			}
		case exclusionRuleGroupID:
			if _, exists := groupIDs[value]; value != "" && exists {
				return fmt.Sprintf("excluded by group_id rule %s", rule.ID), nil
			}
		case exclusionRuleGroupName:
			if _, exists := groupNames[value]; value != "" && exists {
				return fmt.Sprintf("excluded by group_name rule %s", rule.ID), nil
			}
		case exclusionRuleIsBot:
			if value == "true" && user.IsBot {
				return fmt.Sprintf("excluded by is_bot rule %s", rule.ID), nil
			}
		case exclusionRuleDeleted:
			if value == "true" && user.DeleteAt != 0 {
				return fmt.Sprintf("excluded by deleted rule %s", rule.ID), nil
			}
		}
	}

	return "", nil
}

func (p *Plugin) renderOnboardingMessage(user *model.User, override *resolvedOnboardingProfile) (onboardingRenderResult, error) {
	config := p.getRuntimeConfiguration()
	profile := p.resolveOnboardingProfile(user, override)
	variables := buildTemplateVariables(user, profile)

	commonTemplate := selectTemplate(config.Templates, templateTypeCommon, "", profile.Language, config.DefaultLanguage)
	if commonTemplate == nil {
		return onboardingRenderResult{}, errNoOnboardingMessage
	}

	departmentCode := profile.DepartmentCode
	if departmentCode == "" {
		departmentCode = config.FallbackDepartmentCode
	}

	departmentTemplate := selectTemplate(config.Templates, templateTypeDepartment, departmentCode, profile.Language, config.DefaultLanguage)
	if departmentTemplate == nil && departmentCode != config.FallbackDepartmentCode {
		departmentTemplate = selectTemplate(config.Templates, templateTypeDepartment, config.FallbackDepartmentCode, profile.Language, config.DefaultLanguage)
	}

	sections := make([]string, 0, 2)
	appliedTemplateIDs := []string{commonTemplate.ID}

	commonSection, commonSkippedLinks := renderTemplateSection(commonTemplate, linksForTemplate(config.Links, commonTemplate.ID), variables)
	if commonSection != "" {
		sections = append(sections, commonSection)
	}

	skippedLinks := commonSkippedLinks
	departmentTemplateID := ""
	if departmentTemplate != nil {
		departmentSection, deptSkippedLinks := renderTemplateSection(departmentTemplate, linksForTemplate(config.Links, departmentTemplate.ID), variables)
		if strings.TrimSpace(departmentSection) != "" {
			sections = append(sections, departmentSection)
			appliedTemplateIDs = append(appliedTemplateIDs, departmentTemplate.ID)
			departmentTemplateID = departmentTemplate.ID
		}
		skippedLinks += deptSkippedLinks
	}

	message := strings.TrimSpace(strings.Join(sections, "\n\n"))
	if message == "" {
		return onboardingRenderResult{}, errNoOnboardingMessage
	}

	return onboardingRenderResult{
		Message:              message,
		Language:             profile.Language,
		DepartmentCode:       departmentCode,
		DepartmentName:       profile.DepartmentName,
		CommonTemplateID:     commonTemplate.ID,
		DepartmentTemplateID: departmentTemplateID,
		AppliedTemplateIDs:   appliedTemplateIDs,
		SkippedLinkCount:     skippedLinks,
	}, nil
}

func (p *Plugin) resolveOnboardingProfile(user *model.User, override *resolvedOnboardingProfile) resolvedOnboardingProfile {
	config := p.getRuntimeConfiguration()
	profile := resolvedOnboardingProfile{
		DepartmentCode: firstNonEmpty(
			getUserProp(user, "department_code"),
			getUserProp(user, "dept_code"),
			getUserProp(user, "department"),
		),
		DepartmentName: firstNonEmpty(
			getUserProp(user, "department_name"),
			getUserProp(user, "department"),
			getUserProp(user, "dept_name"),
		),
		Organization: firstNonEmpty(
			getUserProp(user, "organization_name"),
			getUserProp(user, "organization"),
		),
		Language: normalizeLanguage(firstNonEmpty(user.Locale, config.DefaultLanguage)),
		StartDate: firstNonEmpty(
			getUserProp(user, "start_date"),
			getUserProp(user, "hire_date"),
			getUserProp(user, "join_date"),
		),
	}

	if mapping := matchDepartmentMapping(config.Mappings, profile.DepartmentCode, profile.DepartmentName); mapping != nil {
		if mapping.TemplateDepartmentCode != "" {
			profile.DepartmentCode = mapping.TemplateDepartmentCode
		}
		if mapping.DepartmentName != "" {
			profile.DepartmentName = mapping.DepartmentName
		}
		if mapping.OrganizationName != "" {
			profile.Organization = mapping.OrganizationName
		}
	}

	if override != nil {
		profile.DepartmentCode = firstNonEmpty(override.DepartmentCode, profile.DepartmentCode)
		profile.DepartmentName = firstNonEmpty(override.DepartmentName, profile.DepartmentName)
		profile.Organization = firstNonEmpty(override.Organization, profile.Organization)
		profile.Language = firstNonEmpty(normalizeLanguage(override.Language), profile.Language)
		profile.StartDate = firstNonEmpty(override.StartDate, profile.StartDate)
	}

	if profile.DepartmentName == "" {
		profile.DepartmentName = profile.DepartmentCode
	}
	if profile.Language == "" {
		profile.Language = config.DefaultLanguage
	}

	return profile
}

func matchDepartmentMapping(mappings []departmentMapping, sourceCode, sourceName string) *departmentMapping {
	normalizedCode := strings.ToLower(strings.TrimSpace(sourceCode))
	normalizedName := strings.ToLower(strings.TrimSpace(sourceName))

	for _, mapping := range mappings {
		if !mapping.Active {
			continue
		}
		if normalizedCode != "" && strings.EqualFold(mapping.SourceDepartmentCode, normalizedCode) {
			candidate := mapping
			return &candidate
		}
		if normalizedName != "" && strings.EqualFold(mapping.SourceDepartmentName, normalizedName) {
			candidate := mapping
			return &candidate
		}
	}

	return nil
}

func selectTemplate(templates []onboardingTemplate, templateType, departmentCode, language, defaultLanguage string) *onboardingTemplate {
	var exactMatches []onboardingTemplate
	var defaultLanguageMatches []onboardingTemplate
	var fallbackMatches []onboardingTemplate

	for _, template := range templates {
		if !template.Active || template.Type != templateType {
			continue
		}
		if templateType == templateTypeDepartment && !strings.EqualFold(template.DepartmentCode, departmentCode) {
			continue
		}
		if templateType == templateTypeCommon && template.DepartmentCode != "" {
			continue
		}

		switch {
		case strings.EqualFold(template.Lang, language):
			exactMatches = append(exactMatches, template)
		case strings.EqualFold(template.Lang, defaultLanguage):
			defaultLanguageMatches = append(defaultLanguageMatches, template)
		default:
			fallbackMatches = append(fallbackMatches, template)
		}
	}

	if len(exactMatches) > 0 {
		chosen := chooseLatestTemplate(exactMatches)
		return &chosen
	}
	if len(defaultLanguageMatches) > 0 {
		chosen := chooseLatestTemplate(defaultLanguageMatches)
		return &chosen
	}
	if len(fallbackMatches) > 0 {
		chosen := chooseLatestTemplate(fallbackMatches)
		return &chosen
	}

	return nil
}

func chooseLatestTemplate(templates []onboardingTemplate) onboardingTemplate {
	sort.SliceStable(templates, func(i, j int) bool {
		if templates[i].Version == templates[j].Version {
			return templates[i].ID < templates[j].ID
		}
		return templates[i].Version > templates[j].Version
	})
	return templates[0]
}

func linksForTemplate(links []onboardingLink, templateID string) []onboardingLink {
	filtered := make([]onboardingLink, 0)
	for _, link := range links {
		if !link.Active || link.TemplateID != templateID {
			continue
		}
		filtered = append(filtered, link)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].SortOrder == filtered[j].SortOrder {
			return filtered[i].Title < filtered[j].Title
		}
		return filtered[i].SortOrder < filtered[j].SortOrder
	})

	return filtered
}

func renderTemplateSection(template *onboardingTemplate, links []onboardingLink, variables map[string]string) (string, int) {
	if template == nil {
		return "", 0
	}

	parts := make([]string, 0, 3)
	if title := strings.TrimSpace(applyTemplateVariables(template.Title, variables)); title != "" {
		parts = append(parts, fmt.Sprintf("**%s**", title))
	}
	if body := strings.TrimSpace(applyTemplateVariables(template.Body, variables)); body != "" {
		parts = append(parts, body)
	}

	skippedLinks := 0
	if len(links) > 0 {
		renderedLinks := make([]string, 0, len(links))
		for _, link := range links {
			if strings.TrimSpace(link.URL) == "" {
				skippedLinks++
				continue
			}

			entry := fmt.Sprintf("- [%s](%s)", applyTemplateVariables(link.Title, variables), link.URL)
			if description := strings.TrimSpace(applyTemplateVariables(link.Description, variables)); description != "" {
				entry += " - " + description
			}
			renderedLinks = append(renderedLinks, entry)
		}
		if len(renderedLinks) > 0 {
			parts = append(parts, strings.Join(renderedLinks, "\n"))
		}
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n")), skippedLinks
}

func buildTemplateVariables(user *model.User, profile resolvedOnboardingProfile) map[string]string {
	return map[string]string{
		"user_id":           user.Id,
		"username":          user.Username,
		"user_name":         userDisplayName(user),
		"first_name":        strings.TrimSpace(user.FirstName),
		"last_name":         strings.TrimSpace(user.LastName),
		"email":             strings.TrimSpace(user.Email),
		"department_code":   strings.TrimSpace(profile.DepartmentCode),
		"department_name":   strings.TrimSpace(profile.DepartmentName),
		"organization_name": strings.TrimSpace(profile.Organization),
		"start_date":        strings.TrimSpace(profile.StartDate),
		"language_code":     strings.TrimSpace(profile.Language),
	}
}

func applyTemplateVariables(input string, variables map[string]string) string {
	if strings.TrimSpace(input) == "" {
		return input
	}

	replacements := make([]string, 0, len(variables)*2)
	for key, value := range variables {
		replacements = append(replacements, "{{"+key+"}}", value)
	}

	replacer := strings.NewReplacer(replacements...)
	return replacer.Replace(input)
}

func userDisplayName(user *model.User) string {
	fullName := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if fullName != "" {
		return fullName
	}
	if strings.TrimSpace(user.Nickname) != "" {
		return strings.TrimSpace(user.Nickname)
	}
	return strings.TrimSpace(user.Username)
}

func getUserProp(user *model.User, key string) string {
	if user == nil || user.Props == nil {
		return ""
	}

	return strings.TrimSpace(user.Props[key])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

