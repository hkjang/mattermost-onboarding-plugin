package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const (
	automaticSendStatePrefix = "onboarding/send_state/"
	sendLogPrefix            = "onboarding/send_log/"
	latestSendLogPrefix      = "onboarding/latest_log/"
	queuePrefix              = "onboarding/queue/"
)

func automaticSendStateKey(userID string) string {
	return automaticSendStatePrefix + userID
}

func sendLogKey(logID string) string {
	return sendLogPrefix + logID
}

func latestSendLogKey(userID string) string {
	return latestSendLogPrefix + userID
}

func queueItemKey(userID string) string {
	return queuePrefix + userID
}

func (p *Plugin) claimAutomaticSend(userID string) (sendState, bool, error) {
	state := sendState{
		Status:    "processing",
		UpdatedAt: time.Now().UTC().UnixMilli(),
	}

	stored, err := p.client.KV.Set(
		automaticSendStateKey(userID),
		state,
		pluginapi.SetAtomic(nil),
		pluginapi.SetExpiry(15*time.Minute),
	)
	if err != nil {
		return sendState{}, false, fmt.Errorf("failed to claim automatic send state: %w", err)
	}

	return state, stored, nil
}

func (p *Plugin) finalizeAutomaticSend(userID string, state sendState) error {
	if _, err := p.client.KV.Set(automaticSendStateKey(userID), state); err != nil {
		return fmt.Errorf("failed to finalize automatic send state: %w", err)
	}

	return nil
}

func (p *Plugin) releaseAutomaticSend(userID string) error {
	if err := p.client.KV.Delete(automaticSendStateKey(userID)); err != nil {
		return fmt.Errorf("failed to release automatic send state: %w", err)
	}

	return nil
}

func (p *Plugin) getAutomaticSendState(userID string) (*sendState, error) {
	var state sendState
	if err := p.client.KV.Get(automaticSendStateKey(userID), &state); err != nil {
		return nil, fmt.Errorf("failed to load automatic send state: %w", err)
	}

	if state.Status == "" && state.UpdatedAt == 0 && state.LogID == "" {
		return nil, nil
	}

	return &state, nil
}

func (p *Plugin) saveSendLog(log *onboardingSendLog) error {
	if log == nil {
		return nil
	}

	if _, err := p.client.KV.Set(sendLogKey(log.ID), log); err != nil {
		return fmt.Errorf("failed to save send log history: %w", err)
	}
	if _, err := p.client.KV.Set(latestSendLogKey(log.UserID), log); err != nil {
		return fmt.Errorf("failed to save latest send log: %w", err)
	}

	return nil
}

func (p *Plugin) getLatestSendLog(userID string) (*onboardingSendLog, error) {
	var log onboardingSendLog
	if err := p.client.KV.Get(latestSendLogKey(userID), &log); err != nil {
		return nil, fmt.Errorf("failed to read latest send log: %w", err)
	}

	if log.ID == "" {
		return nil, nil
	}

	return &log, nil
}

func (p *Plugin) listSendLogs(limit int) ([]onboardingSendLog, error) {
	if limit <= 0 {
		limit = 100
	}

	keys, err := p.client.KV.ListKeys(0, limit, pluginapi.WithPrefix(sendLogPrefix))
	if err != nil {
		return nil, fmt.Errorf("failed to list send log keys: %w", err)
	}

	logs := make([]onboardingSendLog, 0, len(keys))
	for _, key := range keys {
		var log onboardingSendLog
		if getErr := p.client.KV.Get(key, &log); getErr != nil {
			return nil, fmt.Errorf("failed to load send log %s: %w", key, getErr)
		}
		if log.ID != "" {
			logs = append(logs, log)
		}
	}

	sort.SliceStable(logs, func(i, j int) bool {
		return logs[i].SentAt > logs[j].SentAt
	})

	return logs, nil
}

func (p *Plugin) saveQueueItem(item *onboardingQueueItem) error {
	if item == nil {
		return nil
	}

	if _, err := p.client.KV.Set(queueItemKey(item.TargetUserID), item); err != nil {
		return fmt.Errorf("failed to save queue item: %w", err)
	}

	return nil
}

func (p *Plugin) getQueueItem(userID string) (*onboardingQueueItem, error) {
	var item onboardingQueueItem
	if err := p.client.KV.Get(queueItemKey(userID), &item); err != nil {
		return nil, fmt.Errorf("failed to read queue item: %w", err)
	}

	if item.TargetUserID == "" {
		return nil, nil
	}

	return &item, nil
}

func (p *Plugin) deleteQueueItem(userID string) error {
	if err := p.client.KV.Delete(queueItemKey(userID)); err != nil {
		return fmt.Errorf("failed to delete queue item: %w", err)
	}

	return nil
}

func (p *Plugin) listDueQueueItems(limit int, now time.Time) ([]onboardingQueueItem, error) {
	if limit <= 0 {
		limit = 100
	}

	keys, err := p.client.KV.ListKeys(0, limit, pluginapi.WithPrefix(queuePrefix))
	if err != nil {
		return nil, fmt.Errorf("failed to list queue keys: %w", err)
	}

	items := make([]onboardingQueueItem, 0, len(keys))
	nowMillis := now.UTC().UnixMilli()
	for _, key := range keys {
		var item onboardingQueueItem
		if getErr := p.client.KV.Get(key, &item); getErr != nil {
			return nil, fmt.Errorf("failed to load queue item %s: %w", key, getErr)
		}
		if item.TargetUserID == "" {
			continue
		}
		if item.Status != queueStatusPending {
			continue
		}
		if item.NextAttemptAt > nowMillis {
			continue
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].NextAttemptAt < items[j].NextAttemptAt
	})

	return items, nil
}
