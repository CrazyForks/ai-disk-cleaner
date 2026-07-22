package i18n

import (
	"embed"
	"encoding/json"
	"path"
	"strings"
)

const defaultLanguage = "en"

//go:embed locales/*.json
var localeFiles embed.FS

type localeResource struct {
	UserPrompt string `json:"userPrompt"`
}

// AnalyzerUserPrompt returns the localized initial request for disk analysis.
func AnalyzerUserPrompt(language string) string {
	resource, ok := loadLocaleResource(normalizeLanguage(language))
	if !ok {
		resource, ok = loadLocaleResource(defaultLanguage)
	}
	if !ok {
		panic("i18n: default locale resource is missing or invalid")
	}
	return resource.UserPrompt
}

func normalizeLanguage(language string) string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(language), "_", "-"))
	if normalized == "zh" || strings.HasPrefix(normalized, "zh-") {
		return "zh_CN"
	}
	return defaultLanguage
}

func loadLocaleResource(language string) (localeResource, bool) {
	contents, err := localeFiles.ReadFile(path.Join("locales", language+".json"))
	if err != nil {
		return localeResource{}, false
	}
	var resource localeResource
	if err := json.Unmarshal(contents, &resource); err != nil || strings.TrimSpace(resource.UserPrompt) == "" {
		return localeResource{}, false
	}
	return resource, true
}
