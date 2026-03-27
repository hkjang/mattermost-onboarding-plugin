package main

import (
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/pkg/errors"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// client is the Mattermost server API client.
	client *pluginapi.Client

	// router is the HTTP router for handling API requests.
	router *mux.Router

	backgroundJob *cluster.Job
	botUserID     string
	botLock       sync.RWMutex

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)
	p.router = p.initRouter()

	if err := p.OnConfigurationChange(); err != nil {
		return err
	}

	job, err := cluster.Schedule(
		p.API,
		"OnboardingRetryJob",
		cluster.MakeWaitForRoundedInterval(1*time.Minute),
		p.runJob,
	)
	if err != nil {
		return errors.Wrap(err, "failed to schedule background job")
	}

	p.backgroundJob = job

	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	if p.backgroundJob != nil {
		if err := p.backgroundJob.Close(); err != nil {
			p.API.LogError("Failed to close background job", "err", err)
		}
	}
	return nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/

func (p *Plugin) ensureSenderBot() error {
	config := p.getRuntimeConfiguration()

	botUserID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    config.SenderBotUsername,
		DisplayName: config.SenderBotDisplayName,
		Description: "Automatically sends onboarding guidance to newly created users.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure sender bot")
	}

	p.botLock.Lock()
	defer p.botLock.Unlock()
	p.botUserID = botUserID

	return nil
}

func (p *Plugin) getBotUserID() string {
	p.botLock.RLock()
	defer p.botLock.RUnlock()
	return p.botUserID
}
