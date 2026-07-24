package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuanli27/octopus/internal/op"
	"github.com/xuanli27/octopus/internal/relay/balancer"
	"github.com/xuanli27/octopus/internal/server/middleware"
	"github.com/xuanli27/octopus/internal/server/resp"
	"github.com/xuanli27/octopus/internal/server/router"
)

func init() {
	router.NewGroupRouter("/api/v1/runtime").
		Use(middleware.Auth()).
		AddRoute(router.NewRoute("/overview", http.MethodGet).Handle(getRuntimeOverview))
}

type runtimeCircuitView struct {
	ChannelID           int    `json:"channel_id"`
	ChannelName         string `json:"channel_name"`
	ChannelKeyID        int    `json:"channel_key_id"`
	ModelName           string `json:"model_name"`
	State               string `json:"state"`
	ConsecutiveFailures int64  `json:"consecutive_failures"`
	TripCount           int    `json:"trip_count"`
	RemainingCooldownMS int64  `json:"remaining_cooldown_ms"`
}

type runtimeChannelHealth struct {
	ChannelID      int     `json:"channel_id"`
	ChannelName    string  `json:"channel_name"`
	RequestSuccess int64   `json:"request_success"`
	RequestFailed  int64   `json:"request_failed"`
	TotalRequests  int64   `json:"total_requests"`
	FailRate       float64 `json:"fail_rate"` // 0-100
	Enabled        bool    `json:"enabled"`
	Window         string  `json:"window"` // e.g. "1h"
}

type runtimeOverview struct {
	OpenCircuits     int                    `json:"open_circuits"`
	HalfOpenCircuits int                    `json:"half_open_circuits"`
	Circuits         []runtimeCircuitView   `json:"circuits"`
	ChannelHealth    []runtimeChannelHealth `json:"channel_health"`
	UnhealthyCount   int                    `json:"unhealthy_count"`
	HealthWindow     string                 `json:"health_window"`
}

func getRuntimeOverview(c *gin.Context) {
	snaps := balancer.ListCircuitSnapshots()
	views := make([]runtimeCircuitView, 0, len(snaps))
	open, half := 0, 0
	for _, s := range snaps {
		name := ""
		if ch, err := op.ChannelGet(s.ChannelID, c.Request.Context()); err == nil && ch != nil {
			name = ch.Name
		}
		views = append(views, runtimeCircuitView{
			ChannelID:           s.ChannelID,
			ChannelName:         name,
			ChannelKeyID:        s.ChannelKeyID,
			ModelName:           s.ModelName,
			State:               s.State,
			ConsecutiveFailures: s.ConsecutiveFailures,
			TripCount:           s.TripCount,
			RemainingCooldownMS: s.RemainingCooldownMS,
		})
		switch s.State {
		case "open":
			open++
		case "half_open":
			half++
		}
	}
	sort.Slice(views, func(i, j int) bool {
		if views[i].State == views[j].State {
			if views[i].ChannelName == views[j].ChannelName {
				return views[i].ModelName < views[j].ModelName
			}
			return views[i].ChannelName < views[j].ChannelName
		}
		rank := func(s string) int {
			switch s {
			case "open":
				return 0
			case "half_open":
				return 1
			default:
				return 2
			}
		}
		return rank(views[i].State) < rank(views[j].State)
	})

	const windowLabel = "1h"
	window := time.Hour
	recent := op.StatsChannelRecentSnapshot(window)
	health := make([]runtimeChannelHealth, 0, len(recent))
	unhealthy := 0
	for _, r := range recent {
		name := ""
		enabled := true
		if ch, err := op.ChannelGet(r.ChannelID, c.Request.Context()); err == nil && ch != nil {
			name = ch.Name
			enabled = ch.Enabled
		}
		item := runtimeChannelHealth{
			ChannelID:      r.ChannelID,
			ChannelName:    name,
			RequestSuccess: r.RequestSuccess,
			RequestFailed:  r.RequestFailed,
			TotalRequests:  r.TotalRequests,
			FailRate:       r.FailRate,
			Enabled:        enabled,
			Window:         windowLabel,
		}
		// Surface any channel with failures in the window; count high-fail as unhealthy.
		if r.RequestFailed > 0 || r.FailRate >= 10 {
			health = append(health, item)
		}
		if r.FailRate >= 20 || r.RequestFailed >= 3 {
			unhealthy++
		}
	}
	sort.Slice(health, func(i, j int) bool {
		if health[i].FailRate == health[j].FailRate {
			if health[i].RequestFailed == health[j].RequestFailed {
				return health[i].ChannelName < health[j].ChannelName
			}
			return health[i].RequestFailed > health[j].RequestFailed
		}
		return health[i].FailRate > health[j].FailRate
	})
	if len(health) > 20 {
		health = health[:20]
	}

	resp.Success(c, runtimeOverview{
		OpenCircuits:     open,
		HalfOpenCircuits: half,
		Circuits:         views,
		ChannelHealth:    health,
		UnhealthyCount:   unhealthy,
		HealthWindow:     windowLabel,
	})
}
