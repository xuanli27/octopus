package op

import (
	"testing"

	"github.com/xuanli27/octopus/internal/model"
)

func TestStatsModelNameIDStableAndNonZero(t *testing.T) {
	a := statsModelNameID("gpt-4o")
	b := statsModelNameID("GPT-4O")
	if a == 0 || a != b {
		t.Fatalf("expected stable non-zero id, got %d %d", a, b)
	}
	if statsModelNameID("gpt-4o-mini") == a {
		t.Fatalf("different models should not collide in common cases")
	}
}

func TestStatsModelNameUpdateAggregates(t *testing.T) {
	// use in-memory cache only — no DB
	statsModelCache.Clear()
	statsModelCacheNeedUpdateLock.Lock()
	statsModelCacheNeedUpdate = make(map[int]struct{})
	statsModelCacheNeedUpdateLock.Unlock()

	_ = StatsModelNameUpdate("gpt-4o", 1, model.StatsMetrics{InputToken: 3, CacheReadToken: 1, RequestSuccess: 1})
	_ = StatsModelNameUpdate("gpt-4o", 2, model.StatsMetrics{InputToken: 2, CacheReadToken: 2, RequestSuccess: 1})
	list := StatsModelList()
	if len(list) != 1 {
		t.Fatalf("expected 1 model row, got %#v", list)
	}
	if list[0].InputToken != 5 || list[0].CacheReadToken != 3 || list[0].RequestSuccess != 2 {
		t.Fatalf("aggregate mismatch: %+v", list[0])
	}
	if list[0].Name != "gpt-4o" {
		t.Fatalf("name: %q", list[0].Name)
	}
}
