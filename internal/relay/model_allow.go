package relay

import (
	"strings"

	"github.com/xuanli27/octopus/internal/model"
)

// apiKeyAllowsModel evaluates SupportedModels + ModelListMode the same way as model.APIKey.ModelAllowed.
func apiKeyAllowsModel(supportedModelsCSV, mode, requestModel string) bool {
	key := model.APIKey{
		SupportedModels: supportedModelsCSV,
		ModelListMode:   mode,
	}
	return key.ModelAllowed(strings.TrimSpace(requestModel))
}
