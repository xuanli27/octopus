package op

import (
	"context"
	"fmt"
	"strings"

	"github.com/xuanli27/octopus/internal/db"
	"github.com/xuanli27/octopus/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func PublicModelList(ctx context.Context) ([]model.PublicModel, error) {
	var rows []model.PublicModel
	err := db.GetDB().WithContext(ctx).
		Preload("Aliases", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("alias ASC")
		}).
		Order("name ASC").
		Find(&rows).Error
	return rows, err
}

func PublicModelGet(id int, ctx context.Context) (*model.PublicModel, error) {
	var row model.PublicModel
	err := db.GetDB().WithContext(ctx).
		Preload("Aliases").
		First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func PublicModelGetByName(name string, ctx context.Context) (*model.PublicModel, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row model.PublicModel
	err := db.GetDB().WithContext(ctx).
		Preload("Aliases").
		Where("LOWER(name) = ?", strings.ToLower(name)).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func PublicModelCreate(req *model.PublicModelCreateRequest, ctx context.Context) (*model.PublicModel, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name required")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	aliases := normalizeAliasList(req.Aliases)
	// ensure name itself not required as alias; allow optional

	row := &model.PublicModel{
		Name:    name,
		Enabled: enabled,
		Note:    strings.TrimSpace(req.Note),
	}
	err := db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return replaceAliasesTx(tx, row.ID, aliases)
	})
	if err != nil {
		return nil, err
	}
	return PublicModelGet(row.ID, ctx)
}

func PublicModelUpdate(req *model.PublicModelUpdateRequest, ctx context.Context) (*model.PublicModel, error) {
	if req == nil || req.ID <= 0 {
		return nil, fmt.Errorf("invalid request")
	}
	row, err := PublicModelGet(req.ID, ctx)
	if err != nil {
		return nil, err
	}
	err = db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{}
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return fmt.Errorf("name required")
			}
			updates["name"] = name
		}
		if req.Note != nil {
			updates["note"] = strings.TrimSpace(*req.Note)
		}
		if req.Enabled != nil {
			updates["enabled"] = *req.Enabled
		}
		if len(updates) > 0 {
			if err := tx.Model(&model.PublicModel{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if req.Aliases != nil {
			return replaceAliasesTx(tx, row.ID, normalizeAliasList(req.Aliases))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return PublicModelGet(row.ID, ctx)
}

func PublicModelDelete(id int, ctx context.Context) error {
	return db.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("public_model_id = ?", id).Delete(&model.PublicModelAlias{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.PublicModel{}, id).Error
	})
}

func replaceAliasesTx(tx *gorm.DB, publicModelID int, aliases []string) error {
	if err := tx.Where("public_model_id = ?", publicModelID).Delete(&model.PublicModelAlias{}).Error; err != nil {
		return err
	}
	if len(aliases) == 0 {
		return nil
	}
	rows := make([]model.PublicModelAlias, 0, len(aliases))
	seen := map[string]struct{}{}
	for _, a := range aliases {
		lower := strings.ToLower(a)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		rows = append(rows, model.PublicModelAlias{
			PublicModelID: publicModelID,
			Alias:         a,
			AliasLower:    lower,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func normalizeAliasList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		a := strings.TrimSpace(raw)
		if a == "" {
			continue
		}
		lower := strings.ToLower(a)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, a)
	}
	return out
}

// publicAliasIndex maps alias_lower / normalized forms -> public name (enabled only).
type publicAliasIndex struct {
	// exact lower(alias) or lower(public name)
	byExact map[string]string
	// normalize(alias/name) lower -> public name
	byNorm map[string]string
}

func loadPublicAliasIndex(ctx context.Context) (*publicAliasIndex, error) {
	models, err := PublicModelList(ctx)
	if err != nil {
		return nil, err
	}
	idx := &publicAliasIndex{
		byExact: make(map[string]string),
		byNorm:  make(map[string]string),
	}
	for _, m := range models {
		if !m.Enabled {
			continue
		}
		pub := strings.TrimSpace(m.Name)
		if pub == "" {
			continue
		}
		idx.byExact[strings.ToLower(pub)] = pub
		if n := NormalizePublicModelName(pub); n != "" {
			idx.byNorm[strings.ToLower(n)] = pub
		}
		for _, a := range m.Aliases {
			al := strings.TrimSpace(a.Alias)
			if al == "" {
				continue
			}
			idx.byExact[strings.ToLower(al)] = pub
			if n := NormalizePublicModelName(al); n != "" {
				// prefer first registration; exact alias later may override via byExact
				if _, exists := idx.byNorm[strings.ToLower(n)]; !exists {
					idx.byNorm[strings.ToLower(n)] = pub
				}
			}
		}
	}
	return idx, nil
}

// ResolvePublicModelName maps an upstream model id to a public name if possible.
// Priority: alias exact → public name exact → normalize(alias/public) with normalize flag.
func ResolvePublicModelName(upstream string, idx *publicAliasIndex, useNormalize bool) (public string, via string) {
	up := strings.TrimSpace(upstream)
	if up == "" {
		return "", "none"
	}
	if idx != nil {
		if pub, ok := idx.byExact[strings.ToLower(up)]; ok {
			return pub, "exact_alias"
		}
		if useNormalize {
			n := NormalizePublicModelName(up)
			if n != "" {
				if pub, ok := idx.byExact[strings.ToLower(n)]; ok {
					return pub, "normalize"
				}
				if pub, ok := idx.byNorm[strings.ToLower(n)]; ok {
					return pub, "normalize"
				}
			}
		}
	}
	// Fall back: pure normalize equals itself (for create-missing without dict)
	if useNormalize {
		n := NormalizePublicModelName(up)
		if n != "" {
			return n, "normalize_local"
		}
	}
	return "", "none"
}

// PublicModelResolveBatch resolves many upstream names for UI.
func PublicModelResolveBatch(upstreams []string, ctx context.Context) ([]model.PublicModelResolveResult, error) {
	idx, err := loadPublicAliasIndex(ctx)
	if err != nil {
		return nil, err
	}
	useNorm := AutoGroupNormalizeEnabled()
	out := make([]model.PublicModelResolveResult, 0, len(upstreams))
	for _, u := range upstreams {
		pub, via := ResolvePublicModelName(u, idx, useNorm)
		if via == "normalize_local" && idx != nil && len(idx.byExact) > 0 {
			// When dictionary exists, treat normalize_local without dict hit as unresolved
			// for UI "pending" lists (avoid false confidence). create-missing still uses full resolve.
		}
		out = append(out, model.PublicModelResolveResult{
			Upstream: u,
			Public:   pub,
			Via:      via,
		})
	}
	return out, nil
}


// InferModelFamily maps a public/upstream name to a coarse vendor family for UI filters.
// This is intentionally a small built-in list (not DB): stable defaults, no migration sprawl.
// Users still organize via public names/aliases; families are for browsing only.
func InferModelFamily(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	if s == "" {
		return "other"
	}
	// order matters for overlapping prefixes
	switch {
	case strings.Contains(s, "claude"):
		return "claude"
	case strings.Contains(s, "gpt") || strings.Contains(s, "o1") || strings.Contains(s, "o3") || strings.Contains(s, "o4") || strings.HasPrefix(s, "chatgpt"):
		return "openai"
	case strings.Contains(s, "deepseek"):
		return "deepseek"
	case strings.Contains(s, "gemini") || strings.Contains(s, "gemma"):
		return "google"
	case strings.Contains(s, "qwen") || strings.Contains(s, "qwq"):
		return "qwen"
	case strings.Contains(s, "glm") || strings.Contains(s, "chatglm"):
		return "zhipu"
	case strings.Contains(s, "moonshot") || strings.Contains(s, "kimi"):
		return "moonshot"
	case strings.Contains(s, "minimax") || strings.Contains(s, "abab"):
		return "minimax"
	case strings.Contains(s, "mistral") || strings.Contains(s, "mixtral") || strings.Contains(s, "codestral"):
		return "mistral"
	case strings.Contains(s, "llama") || strings.Contains(s, "meta-llama"):
		return "meta"
	case strings.Contains(s, "grok"):
		return "xai"
	case strings.Contains(s, "command-r") || strings.Contains(s, "cohere"):
		return "cohere"
	default:
		return "other"
	}
}


// PublicModelListPending scans enabled channels and returns upstream models that do not
// resolve via dictionary exact alias / public name (normalize-only suggestions still pending).
func PublicModelListPending(ctx context.Context) ([]model.PublicModelPendingItem, error) {
	channels, err := ChannelList(ctx)
	if err != nil {
		return nil, err
	}
	idx, err := loadPublicAliasIndex(ctx)
	if err != nil {
		return nil, err
	}
	useNorm := AutoGroupNormalizeEnabled()
	seen := map[string]struct{}{}
	out := make([]model.PublicModelPendingItem, 0)
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		names := splitChannelModelNames(ch.Model, ch.CustomModel)
		for _, up := range names {
			up = strings.TrimSpace(up)
			if up == "" {
				continue
			}
			key := strings.ToLower(up)
			if _, ok := seen[key]; ok {
				continue
			}
			pub, via := ResolvePublicModelName(up, idx, useNorm)
			// Resolved only if dictionary-backed (exact_alias / normalize hit on dict).
			if via == "exact_alias" || via == "normalize" {
				continue
			}
			seen[key] = struct{}{}
			item := model.PublicModelPendingItem{
				Upstream:    up,
				ChannelID:   ch.ID,
				ChannelName: ch.Name,
			}
			if via == "normalize_local" && pub != "" {
				item.SuggestedPublic = pub
				item.Via = via
			}
			out = append(out, item)
		}
	}
	return out, nil
}

// PublicModelSeedCommon inserts a small built-in dictionary if names are missing.
func PublicModelSeedCommon(ctx context.Context) (created int, err error) {
	type seed struct {
		name    string
		aliases []string
	}
	seeds := []seed{
		{name: "gpt-4o", aliases: []string{"openai/gpt-4o", "gpt-4o-2024-08-06", "gpt-4o-2024-11-20", "chatgpt-4o-latest"}},
		{name: "gpt-4o-mini", aliases: []string{"openai/gpt-4o-mini", "gpt-4o-mini-2024-07-18"}},
		{name: "gpt-4.1", aliases: []string{"openai/gpt-4.1", "gpt-4-1"}},
		{name: "gpt-4.1-mini", aliases: []string{"openai/gpt-4.1-mini", "gpt-4-1-mini"}},
		{name: "o1", aliases: []string{"openai/o1", "o1-2024-12-17"}},
		{name: "o3-mini", aliases: []string{"openai/o3-mini"}},
		{name: "claude-3.5-sonnet", aliases: []string{"claude-3-5-sonnet", "claude-3-5-sonnet-20241022", "anthropic/claude-3.5-sonnet", "anthropic/claude-3-5-sonnet-20241022"}},
		{name: "claude-3.5-haiku", aliases: []string{"claude-3-5-haiku", "claude-3-5-haiku-20241022", "anthropic/claude-3.5-haiku"}},
		{name: "claude-sonnet-4", aliases: []string{"claude-sonnet-4-20250514", "anthropic/claude-sonnet-4"}},
		{name: "claude-opus-4", aliases: []string{"claude-opus-4-20250514", "anthropic/claude-opus-4"}},
		{name: "deepseek-chat", aliases: []string{"deepseek/deepseek-chat", "deepseek-v3"}},
		{name: "deepseek-reasoner", aliases: []string{"deepseek/deepseek-reasoner", "deepseek-r1"}},
		{name: "gemini-2.0-flash", aliases: []string{"google/gemini-2.0-flash", "gemini-2.0-flash-001"}},
		{name: "gemini-2.5-pro", aliases: []string{"google/gemini-2.5-pro", "gemini-2.5-pro-preview"}},
		{name: "qwen-max", aliases: []string{"qwen/qwen-max", "qwen-max-latest"}},
		{name: "qwen-plus", aliases: []string{"qwen/qwen-plus"}},
	}
	for _, s := range seeds {
		if _, err := PublicModelGetByName(s.name, ctx); err == nil {
			continue
		}
		if _, err := PublicModelCreate(&model.PublicModelCreateRequest{
			Name:    s.name,
			Aliases: s.aliases,
		}, ctx); err != nil {
			// ignore unique races
			if !strings.Contains(strings.ToLower(err.Error()), "unique") {
				return created, err
			}
			continue
		}
		created++
	}
	return created, nil
}
