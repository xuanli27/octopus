package op

import (
	"path/filepath"
	"testing"

	dbpkg "github.com/xuanli27/octopus/internal/db"
	"github.com/xuanli27/octopus/internal/model"
	"github.com/xuanli27/octopus/internal/transformer/outbound"
)

func setupAutoGroupTestDB(t *testing.T) {
	t.Helper()
	if dbpkg.GetDB() != nil {
		_ = dbpkg.Close()
	}
	dbPath := filepath.Join(t.TempDir(), "octopus-auto-group-test.db")
	if err := dbpkg.InitDB("sqlite", dbPath, false); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	t.Cleanup(func() { _ = dbpkg.Close() })

	// Clear in-process caches between tests (package-level state).
	groupCache.Clear()
	groupMap.Clear()
	channelCache.Clear()
}

func groupModelNames(t *testing.T, groupID, channelID int) []string {
	t.Helper()
	group, err := GroupGet(groupID, t.Context())
	if err != nil {
		t.Fatalf("GroupGet: %v", err)
	}
	out := make([]string, 0)
	for _, item := range group.Items {
		if item.ChannelID == channelID {
			out = append(out, item.ModelName)
		}
	}
	return out
}

func hasAllModels(have []string, want ...string) bool {
	set := make(map[string]struct{}, len(have))
	for _, h := range have {
		set[h] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return false
		}
	}
	return true
}

func testChannel(name, models string, auto model.AutoGroupType) *model.Channel {
	return &model.Channel{
		Name:      name,
		Type:      outbound.OutboundTypeOpenAIChat,
		Enabled:   true,
		BaseUrls:  []model.BaseUrl{{URL: "https://example.com"}},
		Model:     models,
		AutoGroup: auto,
		Keys:      []model.ChannelKey{{ChannelKey: "sk-test", Enabled: true}},
	}
}

func TestChannelAutoGroupRegexPrunesNonMatchingModels(t *testing.T) {
	// Issue #105: site sync previously only BatchAdd'd matches; non-matching
	// models already in the group (or fuzzy leftovers) were never removed.
	setupAutoGroupTestDB(t)
	ctx := t.Context()

	channel := testChannel("ch-regex", "gpt-5.4,gpt-5.4-mini,gpt-5.4-nano,claude-sonnet-4", model.AutoGroupTypeRegex)
	if err := ChannelCreate(channel, ctx); err != nil {
		t.Fatalf("ChannelCreate: %v", err)
	}

	group := &model.Group{
		Name:       "gpt-5.4",
		MatchRegex: `^gpt-5\.4(-pro)?$`,
		Mode:       model.GroupModeRoundRobin,
	}
	if err := GroupCreate(group, ctx); err != nil {
		t.Fatalf("GroupCreate: %v", err)
	}

	// Seed with the bad set users report: mini/nano wrongly present.
	if err := GroupItemBatchAdd(group.ID, []model.GroupIDAndLLMName{
		{ChannelID: channel.ID, ModelName: "gpt-5.4"},
		{ChannelID: channel.ID, ModelName: "gpt-5.4-mini"},
		{ChannelID: channel.ID, ModelName: "gpt-5.4-nano"},
	}, ctx); err != nil {
		t.Fatalf("seed GroupItemBatchAdd: %v", err)
	}

	ch, err := ChannelGet(channel.ID, ctx)
	if err != nil {
		t.Fatalf("ChannelGet: %v", err)
	}
	ChannelAutoGroupWithMode(ch, model.AutoGroupTypeRegex, ctx)

	got := groupModelNames(t, group.ID, channel.ID)
	if !hasAllModels(got, "gpt-5.4") {
		t.Fatalf("expected gpt-5.4 kept, got %v", got)
	}
	for _, bad := range []string{"gpt-5.4-mini", "gpt-5.4-nano", "claude-sonnet-4"} {
		if hasAllModels(got, bad) {
			t.Fatalf("expected %q pruned by regex, got %v", bad, got)
		}
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 model after reconcile, got %v", got)
	}
}

func TestChannelAutoGroupRegexReAddsMatchingAfterSync(t *testing.T) {
	setupAutoGroupTestDB(t)
	ctx := t.Context()

	channel := testChannel("ch-readd", "gpt-5.4,gpt-5.4-pro", model.AutoGroupTypeRegex)
	if err := ChannelCreate(channel, ctx); err != nil {
		t.Fatalf("ChannelCreate: %v", err)
	}

	group := &model.Group{
		Name:       "gpt-5.4",
		MatchRegex: `^gpt-5\.4(-pro)?$`,
		Mode:       model.GroupModeRoundRobin,
	}
	if err := GroupCreate(group, ctx); err != nil {
		t.Fatalf("GroupCreate: %v", err)
	}

	ch, _ := ChannelGet(channel.ID, ctx)
	ChannelAutoGroupWithMode(ch, model.AutoGroupTypeRegex, ctx)

	got := groupModelNames(t, group.ID, channel.ID)
	if !hasAllModels(got, "gpt-5.4", "gpt-5.4-pro") || len(got) != 2 {
		t.Fatalf("expected both matching models added, got %v", got)
	}
}

func TestChannelAutoGroupDoesNotTouchOtherChannels(t *testing.T) {
	setupAutoGroupTestDB(t)
	ctx := t.Context()

	ch1 := testChannel("ch-1", "gpt-5.4-mini", model.AutoGroupTypeRegex)
	ch2 := testChannel("ch-2", "gpt-5.4", model.AutoGroupTypeRegex)
	if err := ChannelCreate(ch1, ctx); err != nil {
		t.Fatalf("ChannelCreate ch1: %v", err)
	}
	if err := ChannelCreate(ch2, ctx); err != nil {
		t.Fatalf("ChannelCreate ch2: %v", err)
	}

	group := &model.Group{
		Name: "gpt-5.4", MatchRegex: `^gpt-5\.4$`, Mode: model.GroupModeRoundRobin,
	}
	if err := GroupCreate(group, ctx); err != nil {
		t.Fatalf("GroupCreate: %v", err)
	}

	// Manually put mini under ch1 (would not match regex).
	if err := GroupItemBatchAdd(group.ID, []model.GroupIDAndLLMName{
		{ChannelID: ch1.ID, ModelName: "gpt-5.4-mini"},
		{ChannelID: ch2.ID, ModelName: "gpt-5.4"},
	}, ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Reconcile only ch2 — ch1's item must remain (auto-group runs per channel).
	c2, _ := ChannelGet(ch2.ID, ctx)
	ChannelAutoGroupWithMode(c2, model.AutoGroupTypeRegex, ctx)

	if got := groupModelNames(t, group.ID, ch1.ID); !hasAllModels(got, "gpt-5.4-mini") {
		t.Fatalf("other channel items must be preserved, got %v", got)
	}
	if got := groupModelNames(t, group.ID, ch2.ID); !hasAllModels(got, "gpt-5.4") || len(got) != 1 {
		t.Fatalf("ch2 should keep matching model only, got %v", got)
	}

	// Reconcile ch1 with regex → mini should be pruned.
	c1, _ := ChannelGet(ch1.ID, ctx)
	ChannelAutoGroupWithMode(c1, model.AutoGroupTypeRegex, ctx)
	if got := groupModelNames(t, group.ID, ch1.ID); len(got) != 0 {
		t.Fatalf("ch1 non-matching models should be pruned, got %v", got)
	}
}

func TestMatchModelsForAutoGroupModes(t *testing.T) {
	group := model.Group{Name: "gpt-4o", MatchRegex: `^gpt-4o(-mini)?$`}
	models := []string{"gpt-4o", "gpt-4o-mini", "gpt-4.1", "claude-3"}

	exact, ok := matchModelsForAutoGroup(model.AutoGroupTypeExact, group, models, 1)
	if !ok || !hasAllModels(exact, "gpt-4o") || len(exact) != 1 {
		t.Fatalf("exact: got %v ok=%v", exact, ok)
	}

	fuzzy, ok := matchModelsForAutoGroup(model.AutoGroupTypeFuzzy, group, models, 1)
	if !ok || !hasAllModels(fuzzy, "gpt-4o", "gpt-4o-mini") || len(fuzzy) != 2 {
		t.Fatalf("fuzzy: got %v ok=%v", fuzzy, ok)
	}

	regex, ok := matchModelsForAutoGroup(model.AutoGroupTypeRegex, group, models, 1)
	if !ok || !hasAllModels(regex, "gpt-4o", "gpt-4o-mini") || len(regex) != 2 {
		t.Fatalf("regex: got %v ok=%v", regex, ok)
	}

	// Broken regex must not wipe membership (ok=false).
	_, ok = matchModelsForAutoGroup(model.AutoGroupTypeRegex, model.Group{Name: "x", MatchRegex: "("}, models, 1)
	if ok {
		t.Fatalf("invalid regex should return ok=false")
	}
}
