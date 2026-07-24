package op

import (
	"testing"
	"time"
)

func TestStatsChannelRecentSnapshotWindow(t *testing.T) {
	// isolate by using high channel ids unlikely used elsewhere in package tests
	const id = 910001
	recentChannelHealth.Delete(id)

	StatsChannelRecentRecord(id, true)
	StatsChannelRecentRecord(id, true)
	StatsChannelRecentRecord(id, false)

	// inject an old failure outside window
	val, _ := recentChannelHealth.Load(id)
	ring := val.(*recentHealthRing)
	ring.mu.Lock()
	ring.buf[ring.head] = recentHealthSample{at: time.Now().Add(-2 * time.Hour), ok: false}
	ring.head = (ring.head + 1) % recentHealthCap
	if ring.size < recentHealthCap {
		ring.size++
	}
	ring.mu.Unlock()

	snap := StatsChannelRecentSnapshot(time.Hour)
	var found *ChannelRecentHealth
	for i := range snap {
		if snap[i].ChannelID == id {
			found = &snap[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected channel %d in snapshot", id)
	}
	if found.RequestSuccess != 2 || found.RequestFailed != 1 {
		t.Fatalf("expected 2 success / 1 failed in 1h window, got success=%d failed=%d", found.RequestSuccess, found.RequestFailed)
	}
	if found.TotalRequests != 3 {
		t.Fatalf("total=%d", found.TotalRequests)
	}
}
