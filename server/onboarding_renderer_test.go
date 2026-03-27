package main

import (
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
)

func TestRenderOnboardingMessageCombinesCommonAndDepartmentTemplates(t *testing.T) {
	assert := assert.New(t)

	plugin := Plugin{
		configuration: &configuration{
			runtime: &runtimeConfiguration{
				DefaultLanguage:        "ko",
				FallbackDepartmentCode: "DEFAULT",
				Templates: []onboardingTemplate{
					{
						ID:      "common-ko",
						Type:    templateTypeCommon,
						Lang:    "ko",
						Title:   "Welcome",
						Body:    "{{user_name}}, please review the shared onboarding guide.",
						Active:  true,
						Version: 1,
					},
					{
						ID:             "dept-it-ko",
						Type:           templateTypeDepartment,
						DepartmentCode: "IT",
						Lang:           "ko",
						Title:          "IT guide",
						Body:           "{{department_name}} runbook is available below.",
						Active:         true,
						Version:        2,
					},
				},
				Links: []onboardingLink{
					{
						ID:         "link-common",
						TemplateID: "common-ko",
						Title:      "Shared wiki",
						URL:        "https://confluence.example.com/common",
						Active:     true,
					},
					{
						ID:          "link-dept",
						TemplateID:  "dept-it-ko",
						Title:       "IT runbook",
						Description: "{{department_name}} operating notes",
						URL:         "https://confluence.example.com/it",
						Active:      true,
					},
				},
				Mappings: []departmentMapping{
					{
						SourceDepartmentCode:   "it-platform",
						TemplateDepartmentCode: "IT",
						DepartmentName:         "IT Platform",
						Active:                 true,
					},
				},
			},
		},
	}

	user := &model.User{
		Id:        "user-1",
		Username:  "honggildong",
		FirstName: "Gil",
		LastName:  "Dong",
		Locale:    "ko",
		Props: map[string]string{
			"department_code": "it-platform",
		},
	}

	rendered, err := plugin.renderOnboardingMessage(user, nil)
	assert.NoError(err)
	assert.Equal([]string{"common-ko", "dept-it-ko"}, rendered.AppliedTemplateIDs)
	assert.Equal("IT", rendered.DepartmentCode)
	assert.Equal("IT Platform", rendered.DepartmentName)
	assert.True(strings.Contains(rendered.Message, "Gil Dong"))
	assert.True(strings.Contains(rendered.Message, "IT Platform runbook is available below."))
	assert.True(strings.Contains(rendered.Message, "https://confluence.example.com/common"))
	assert.True(strings.Contains(rendered.Message, "https://confluence.example.com/it"))
}

func TestRenderOnboardingMessageFallsBackToCommonTemplate(t *testing.T) {
	assert := assert.New(t)

	plugin := Plugin{
		configuration: &configuration{
			runtime: &runtimeConfiguration{
				DefaultLanguage:        "ko",
				FallbackDepartmentCode: "DEFAULT",
				Templates: []onboardingTemplate{
					{
						ID:      "common-ko",
						Type:    templateTypeCommon,
						Lang:    "ko",
						Title:   "Welcome",
						Body:    "Shared onboarding notice.",
						Active:  true,
						Version: 1,
					},
				},
			},
		},
	}

	user := &model.User{
		Id:       "user-2",
		Username: "jane",
		Locale:   "ko",
	}

	rendered, err := plugin.renderOnboardingMessage(user, nil)
	assert.NoError(err)
	assert.Equal("common-ko", rendered.CommonTemplateID)
	assert.Equal("", rendered.DepartmentTemplateID)
	assert.Equal([]string{"common-ko"}, rendered.AppliedTemplateIDs)
	assert.True(strings.Contains(rendered.Message, "Shared onboarding notice."))
}

func TestBuildRuntimeConfigurationAppliesDefaults(t *testing.T) {
	assert := assert.New(t)

	runtime, err := buildRuntimeConfiguration(&configuration{})
	assert.NoError(err)
	assert.True(runtime.RetryIntervalMinutes > 0)
	assert.True(runtime.RetryMaxAttempts > 0)
	assert.Equal("ko", runtime.DefaultLanguage)
	assert.Equal("DEFAULT", runtime.FallbackDepartmentCode)
	assert.NotEmpty(runtime.Templates)
	assert.NotEmpty(runtime.Links)
}
