package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// initRouter builds the plugin's HTTP API. Two families of routes exist:
//   - Routes under the authenticated subrouter require a valid Mattermost session
//     (a "Mattermost-User-Id" header set by the server) and are used by the webapp.
//   - Routes under the callback subrouter are meant for a server-to-server integration (such as
//     an n8n workflow) and authenticate via a signed URL or shared bearer secret instead, since
//     the caller has no Mattermost session.
func (p *Plugin) initRouter() *mux.Router {
	router := mux.NewRouter()

	authed := router.PathPrefix("/api/v1").Subrouter()
	authed.Use(p.mattermostAuthRequired)
	authed.HandleFunc("/config", p.handleGetConfig).Methods(http.MethodGet)
	authed.HandleFunc("/voice-messages", p.handleCreateVoiceMessage).Methods(http.MethodPost)

	callbacks := router.PathPrefix("/api/v1").Subrouter()
	callbacks.HandleFunc("/transcriptions", p.handleTranscriptionCallback).Methods(http.MethodPost)
	callbacks.HandleFunc("/files/{fileId}", p.handleSignedFileDownload).Methods(http.MethodGet)

	return router
}

// ServeHTTP routes every HTTP request destined for this plugin.
func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.router.ServeHTTP(w, r)
}

func (p *Plugin) mattermostAuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Mattermost-User-Id") == "" {
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
