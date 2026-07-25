package op

import (
	"strings"
	"testing"

	"github.com/xuanli27/octopus/internal/model"
)

func TestPublicModelAliasResolvesAndAutoGroups(t *testing.T) {
	setupAutoGroupTestDB(t)
	ctx := t.Context()

	if err := SettingSetString(model.SettingKeyAutoGroupNormalizeEnabled, "true"); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if err := SettingSetString(model.SettingKeyAutoGroupCreateMissingEnabled, "true"); err != nil {
		t.Fatalf("create missing: %v", err)
	}

	pm, err := PublicModelCreate(&model.PublicModelCreateRequest{
		Name:    "claude-3.5-sonnet",
		Aliases: []string{"claude-3-5-sonnet-20241022", "anthropic/claude-3.5-sonnet"},
	}, ctx)
	if err != nil {
		t.Fatalf("PublicModelCreate: %v", err)
	}
	if pm.Name != "claude-3.5-sonnet" || len(pm.Aliases) != 2 {
		t.Fatalf("unexpected public model: %+v", pm)
	}

	// Seed an existing group for the public name and attach via auto-group
	if err := GroupCreate(&model.Group{Name: "claude-3.5-sonnet", Mode: model.GroupModeRoundRobin}, ctx); err != nil {
		t.Fatalf("GroupCreate: %v", err)
	}

	ch := testChannel("alias-ch", "anthropic/claude-3-5-sonnet-20241022,gpt-4o-2024-08-06", model.AutoGroupTypeExact)
	if err := ChannelCreate(ch, ctx); err != nil {
		t.Fatalf("ChannelCreate: %v", err)
	}
	loaded, _ := ChannelGet(ch.ID, ctx)
	ChannelAutoGroupWithMode(loaded, model.AutoGroupTypeExact, ctx)

	groups, err := GroupList(ctx)
	if err != nil {
		t.Fatalf("GroupList: %v", err)
	}
	// expect claude group has anthropic upstream id; gpt-4o created by create-missing normalize
	var claudeOK, gptOK bool
	for _, g := range groups {
		names := groupModelNames(t, g.ID, ch.ID)
		switch strings.ToLower(g.Name) {
		case "claude-3.5-sonnet":
			if hasAllModels(names, "anthropic/claude-3-5-sonnet-20241022") {
				claudeOK = true
			}
		case "gpt-4o":
			if hasAllModels(names, "gpt-4o-2024-08-06") {
				gptOK = true
			}
		}
	}
	if !claudeOK {
		t.Fatalf("alias did not attach claude upstream into public group, groups=%+v", groups)
	}
	if !gptOK {
		t.Fatalf("normalize create-missing did not create/attach gpt-4o, groups=%+v", groups)
	}
}

func TestResolvePublicModelNamePriority(t *testing.T) {
	setupAutoGroupTestDB(t)
	ctx := t.Context()
	_, err := PublicModelCreate(&model.PublicModelCreateRequest{
		Name:    "gpt-4o",
		Aliases: []string{"gpt-4o-all", "openai/gpt-4o-fast"},
	}, ctx)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	idx, err := loadPublicAliasIndex(ctx)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	pub, via := ResolvePublicModelName("gpt-4o-all", idx, true)
	if pub != "gpt-4o" || via != "exact_alias" {
		t.Fatalf("got %q via %q", pub, via)
	}
	pub, via = ResolvePublicModelName("openai/gpt-4o-2024-08-06", idx, true)
	if pub != "gpt-4o" {
		t.Fatalf("normalize to dict public expected gpt-4o, got %q via %q", pub, via)
	}
}
