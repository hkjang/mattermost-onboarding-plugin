package main

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// initRouter initializes the HTTP router for the plugin.
func (p *Plugin) initRouter() *mux.Router {
	router := mux.NewRouter()

	// Middleware to require that the user is logged in
	router.Use(p.MattermostAuthorizationRequired)

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/health", p.Health).Methods(http.MethodGet)

	adminRouter := apiRouter.PathPrefix("/admin").Subrouter()
	adminRouter.Use(p.SystemAdminAuthorizationRequired)
	adminRouter.HandleFunc("/config", p.HandleGetConfig).Methods(http.MethodGet)
	adminRouter.HandleFunc("/config", p.HandleSaveConfig).Methods(http.MethodPut)
	adminRouter.HandleFunc("/preview", p.HandlePreview).Methods(http.MethodGet)
	adminRouter.HandleFunc("/resend", p.HandleResend).Methods(http.MethodPost)
	adminRouter.HandleFunc("/stats", p.HandleStats).Methods(http.MethodGet)
	adminRouter.HandleFunc("/logs", p.HandleLogs).Methods(http.MethodGet)

	return router
}

// ServeHTTP handles authenticated plugin API requests mounted at
// <siteUrl>/plugins/<plugin-id>/api/v1/.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

func (p *Plugin) MattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) SystemAdminAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		user, err := p.client.User.Get(userID)
		if err != nil {
			http.Error(w, "Failed to load user", http.StatusInternalServerError)
			return
		}
		if !strings.Contains(user.Roles, "system_admin") {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) Health(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("ok")); err != nil {
		p.API.LogError("Failed to write response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
