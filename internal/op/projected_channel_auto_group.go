package op

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xuanli27/octopus/internal/model"
	"github.com/xuanli27/octopus/internal/utils/log"
	"github.com/dlclark/regexp2"
)

func ProjectedChannelGlobalAutoGroupMode() model.AutoGroupType {
	value, err := SettingGetString(model.SettingKeyProjectedChannelAutoGroupEnabled)
	if err != nil {
		return model.AutoGroupTypeNone
	}
	mode, ok := model.ParseAutoGroupSettingValue(value)
	if !ok {
		return model.AutoGroupTypeNone
	}
	return mode
}

func ProjectedChannelGlobalAutoGroupEnabled() bool {
	return ProjectedChannelGlobalAutoGroupMode() != model.AutoGroupTypeNone
}

func EffectiveProjectedChannelAutoGroup(channel model.Channel) model.AutoGroupType {
	if mode := ProjectedChannelGlobalAutoGroupMode(); mode != model.AutoGroupTypeNone {
		return mode
	}
	return channel.AutoGroup
}

// ChannelAutoGroupWithMode reconciles group membership for a channel against
// the configured auto-group rule (exact / fuzzy / regex).
//
// Declarative sync (issue #105): for each group, desired = models on the
// channel that match the rule; existing items for this channel in that group
// that are not desired are removed, missing ones are added. Manual removals of
// non-matching models therefore survive the next site sync.
func ChannelAutoGroupWithMode(channel *model.Channel, autoGroup model.AutoGroupType, ctx context.Context) {
	if channel == nil || autoGroup == model.AutoGroupTypeNone {
		return
	}
	groups, err := GroupList(ctx)
	if err != nil {
		log.Warnf("get group list failed: %v", err)
		return
	}

	channelModelNames := splitChannelModelNames(channel.Model, channel.CustomModel)

	for _, group := range groups {
		desired, ok := matchModelsForAutoGroup(autoGroup, group, channelModelNames, channel.ID)
		if !ok {
			// Rule unusable (e.g. bad regex) — leave existing items alone.
			continue
		}
		if err := reconcileGroupItemsForChannel(group, channel.ID, desired, ctx); err != nil {
			log.Warnf("auto group reconcile failed (channel=%d group=%d): %v", channel.ID, group.ID, err)
		}
	}
}

// matchModelsForAutoGroup returns channel models that should belong to group
// under the given auto-group mode. ok=false means the rule could not be applied
// (skip reconcile; do not wipe existing membership).
func matchModelsForAutoGroup(autoGroup model.AutoGroupType, group model.Group, channelModelNames []string, channelID int) (matched []string, ok bool) {
	matchedModelNames := make([]string, 0)
	switch autoGroup {
	case model.AutoGroupTypeExact:
		for _, modelName := range channelModelNames {
			if strings.EqualFold(modelName, group.Name) {
				matchedModelNames = append(matchedModelNames, modelName)
			}
		}
		return matchedModelNames, true
	case model.AutoGroupTypeFuzzy:
		groupNameLower := strings.ToLower(strings.TrimSpace(group.Name))
		if groupNameLower == "" {
			return nil, true
		}
		for _, modelName := range channelModelNames {
			if strings.Contains(strings.ToLower(modelName), groupNameLower) {
				matchedModelNames = append(matchedModelNames, modelName)
			}
		}
		return matchedModelNames, true
	case model.AutoGroupTypeRegex:
		if group.MatchRegex == "" {
			for _, modelName := range channelModelNames {
				if strings.EqualFold(modelName, group.Name) {
					matchedModelNames = append(matchedModelNames, modelName)
				}
			}
			return matchedModelNames, true
		}

		re, err := regexp2.Compile(group.MatchRegex, regexp2.ECMAScript)
		if err != nil {
			log.Warnf("compile regex failed (channel=%d group=%d regex=%q): %v", channelID, group.ID, group.MatchRegex, err)
			return nil, false
		}
		re.MatchTimeout = 200 * time.Millisecond
		for _, modelName := range channelModelNames {
			matched, err := re.MatchString(modelName)
			if err != nil {
				log.Warnf("match regex failed (channel=%d group=%d regex=%q model=%q): %v", channelID, group.ID, group.MatchRegex, modelName, err)
				continue
			}
			if matched {
				matchedModelNames = append(matchedModelNames, modelName)
			}
		}
		return matchedModelNames, true
	default:
		return nil, false
	}
}

// reconcileGroupItemsForChannel applies desired set for one channel inside one group.
func reconcileGroupItemsForChannel(group model.Group, channelID int, desired []string, ctx context.Context) error {
	desiredSet := make(map[string]struct{}, len(desired))
	for _, name := range desired {
		desiredSet[name] = struct{}{}
	}

	existing := make([]string, 0)
	for _, item := range group.Items {
		if item.ChannelID != channelID {
			continue
		}
		existing = append(existing, item.ModelName)
	}

	toAdd := make([]model.GroupIDAndLLMName, 0)
	for _, name := range desired {
		found := false
		for _, have := range existing {
			if have == name {
				found = true
				break
			}
		}
		if !found {
			toAdd = append(toAdd, model.GroupIDAndLLMName{ChannelID: channelID, ModelName: name})
		}
	}

	toRemove := make([]string, 0)
	for _, have := range existing {
		if _, ok := desiredSet[have]; !ok {
			toRemove = append(toRemove, have)
		}
	}

	if len(toRemove) > 0 {
		if err := GroupItemBatchDelInGroup(group.ID, channelID, toRemove, ctx); err != nil {
			return err
		}
	}
	if len(toAdd) > 0 {
		if err := GroupItemBatchAdd(group.ID, toAdd, ctx); err != nil {
			return err
		}
	}
	return nil
}

func ChannelAutoGroup(channel *model.Channel, ctx context.Context) {
	if channel == nil {
		return
	}
	ChannelAutoGroupWithMode(channel, channel.AutoGroup, ctx)
}

func AutoGroupAllProjectedChannels(ctx context.Context) error {
	mode := ProjectedChannelGlobalAutoGroupMode()
	if mode == model.AutoGroupTypeNone {
		return nil
	}
	channels := channelCache.GetAll()
	if len(channels) == 0 {
		return nil
	}
	channelIDs := make([]int, 0, len(channels))
	for id := range channels {
		channelIDs = append(channelIDs, id)
	}
	bindingMap, err := SiteChannelBindingMapByChannelIDs(channelIDs, ctx)
	if err != nil {
		return err
	}
	for id, channel := range channels {
		if _, ok := bindingMap[id]; !ok {
			continue
		}
		ChannelAutoGroupWithMode(&channel, mode, ctx)
	}
	return nil
}

func splitChannelModelNames(values ...string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			name := strings.TrimSpace(part)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			result = append(result, name)
		}
	}
	return result
}

func ValidateJSONOverrideObject(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return err
	}
	if _, ok := decoded.(map[string]any); !ok {
		return fmt.Errorf("param_override must be a JSON object")
	}
	return nil
}
