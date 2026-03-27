package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

type resendRequest struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

func (p *Plugin) HandleGetConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, p.getAdminConfiguration())
}

func (p *Plugin) HandleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var request onboardingAdminConfig

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}

	config, err := p.saveAdminConfiguration(request)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, config)
}

type previewResponse struct {
	UserID              string   `json:"user_id"`
	Username            string   `json:"username"`
	Language            string   `json:"language"`
	DepartmentCode      string   `json:"department_code"`
	DepartmentName      string   `json:"department_name"`
	CommonTemplateID    string   `json:"common_template_id"`
	DepartmentTemplateID string   `json:"department_template_id,omitempty"`
	AppliedTemplateIDs  []string `json:"applied_template_ids"`
	Message             string   `json:"message"`
	SkippedLinkCount    int      `json:"skipped_link_count"`
}

func (p *Plugin) HandlePreview(w http.ResponseWriter, r *http.Request) {
	user, err := p.resolveTargetUser(r.URL.Query().Get("user_id"), r.URL.Query().Get("username"))
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	override := &resolvedOnboardingProfile{
		DepartmentCode: strings.TrimSpace(r.URL.Query().Get("dept_code")),
		DepartmentName: strings.TrimSpace(r.URL.Query().Get("dept_name")),
		Organization:   strings.TrimSpace(r.URL.Query().Get("organization_name")),
		Language:       strings.TrimSpace(r.URL.Query().Get("lang")),
		StartDate:      strings.TrimSpace(r.URL.Query().Get("start_date")),
	}

	rendered, err := p.renderOnboardingMessage(user, override)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, previewResponse{
		UserID:               user.Id,
		Username:             user.Username,
		Language:             rendered.Language,
		DepartmentCode:       rendered.DepartmentCode,
		DepartmentName:       rendered.DepartmentName,
		CommonTemplateID:     rendered.CommonTemplateID,
		DepartmentTemplateID: rendered.DepartmentTemplateID,
		AppliedTemplateIDs:   rendered.AppliedTemplateIDs,
		Message:              rendered.Message,
		SkippedLinkCount:     rendered.SkippedLinkCount,
	})
}

func (p *Plugin) HandleResend(w http.ResponseWriter, r *http.Request) {
	var request resendRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}

	user, err := p.resolveTargetUser(request.UserID, request.Username)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	requestedBy := r.Header.Get("Mattermost-User-ID")
	log, sendErr := p.sendOnboardingByUserID(user.Id, sendModeManual, requestedBy)
	if sendErr != nil {
		writeError(w, sendErr.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, log)
}

func (p *Plugin) HandleStats(w http.ResponseWriter, r *http.Request) {
	logs, err := p.listSendLogs(500)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stats := onboardingStats{
		ByDepartment: make(map[string]int64),
	}

	for _, log := range logs {
		stats.TotalSends++
		switch log.Status {
		case sendStatusSent:
			stats.Successful++
		case sendStatusFailed:
			stats.Failed++
			if len(stats.RecentFailures) < 5 {
				stats.RecentFailures = append(stats.RecentFailures, log.Username+": "+log.ErrorMessage)
			}
		case sendStatusSkipped:
			stats.Skipped++
		}

		if log.Mode == sendModeManual {
			stats.ManualResends++
		}

		departmentCode := strings.TrimSpace(log.DepartmentCode)
		if departmentCode == "" {
			departmentCode = "UNMAPPED"
		}
		stats.ByDepartment[departmentCode]++
	}

	writeJSON(w, http.StatusOK, stats)
}

func (p *Plugin) HandleLogs(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID != "" {
		log, err := p.getLatestSendLog(userID)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if log == nil {
			writeError(w, "log not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, log)
		return
	}

	logs, err := p.listSendLogs(100)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

func (p *Plugin) resolveTargetUser(userID, username string) (*model.User, error) {
	userID = strings.TrimSpace(userID)
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")

	if userID != "" {
		return p.client.User.Get(userID)
	}
	if username != "" {
		return p.client.User.GetByUsername(username)
	}

	return nil, errors.New("user_id or username is required")
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, message string, statusCode int) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}
