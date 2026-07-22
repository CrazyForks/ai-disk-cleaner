package setting

import (
	"context"
	"fmt"
	"strconv"

	settingmodel "ai-disk-cleanner/backend/data/models/setting"
)

type settingStore interface {
	ListSettings(context.Context) ([]settingmodel.Setting, error)
	SaveSettings(context.Context, []settingmodel.Setting) error
}

var supportedSettingKeys = map[string]struct{}{
	"llm.secret":       {},
	"llm.url":          {},
	"llm.model":        {},
	"llm.max-token":    {},
	"record.max.count": {},
}

func validateSettings(settings []settingmodel.Setting) error {
	if len(settings) != len(supportedSettingKeys) {
		return fmt.Errorf("all %d settings must be provided", len(supportedSettingKeys))
	}

	seen := make(map[string]struct{}, len(settings))
	for _, item := range settings {
		if _, supported := supportedSettingKeys[item.Key]; !supported {
			return fmt.Errorf("unsupported setting key %q", item.Key)
		}
		if _, duplicate := seen[item.Key]; duplicate {
			return fmt.Errorf("duplicate setting key %q", item.Key)
		}
		seen[item.Key] = struct{}{}
		if item.Key == "llm.max-token" || item.Key == "record.max.count" {
			value, err := strconv.ParseInt(item.Value, 10, 64)
			if err != nil || value <= 0 {
				return fmt.Errorf("%s must be a positive integer", item.Key)
			}
		}
	}
	return nil
}
