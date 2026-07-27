package i18n

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
)

type Locale string

const (
	LocaleZhCN    Locale = "zh-CN"
	LocaleEnUS    Locale = "en-US"
	DefaultLocale        = LocaleZhCN
	EnvLocale            = "APP_LOCALE"
)

type Args map[string]any

//go:embed catalogs/*.json
var catalogFiles embed.FS

var (
	catalogs      map[Locale]map[string]string
	currentLocale atomic.Value
	placeholderRE = regexp.MustCompile(`\{([a-zA-Z][a-zA-Z0-9_]*)\}`)
)

func init() {
	var err error
	catalogs, err = loadCatalogs()
	if err != nil {
		panic(err)
	}
	currentLocale.Store(DefaultLocale)
}

func ParseLocale(value string) (Locale, error) {
	switch strings.TrimSpace(value) {
	case "", string(DefaultLocale):
		return DefaultLocale, nil
	case string(LocaleEnUS):
		return LocaleEnUS, nil
	default:
		return "", fmt.Errorf("%s must be %s or %s", EnvLocale, LocaleZhCN, LocaleEnUS)
	}
}

func Configure(value string) error {
	locale, err := ParseLocale(value)
	if err != nil {
		return err
	}
	currentLocale.Store(locale)
	return nil
}

func ConfigureFromEnv() error {
	return Configure(os.Getenv(EnvLocale))
}

func Current() Locale {
	if locale, ok := currentLocale.Load().(Locale); ok {
		return locale
	}
	return DefaultLocale
}

func Has(key string) bool {
	_, ok := catalogs[Current()][key]
	return ok
}

func KeyForText(message string) (string, bool) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", false
	}
	for _, locale := range []Locale{LocaleZhCN, LocaleEnUS} {
		for key, candidate := range catalogs[locale] {
			if candidate == message {
				return key, true
			}
		}
	}
	return "", false
}

// KeyForCode returns the stable generic key for shared response codes.
// Domain-specific callers should return a more precise registered key when one
// exists.
func KeyForCode(code int32) string {
	switch code {
	case 0:
		return "common.success"
	case 400, 4001:
		return "common.invalid_request"
	case 401:
		return "common.unauthenticated"
	case 403, 4030:
		return "common.forbidden"
	case 404:
		return "common.not_found"
	case 429, 42901, 42902, 42911, 42912, 42921:
		return "common.too_many_requests"
	case 499:
		return "common.canceled"
	case 503:
		return "common.backend_unavailable"
	case 504:
		return "common.timeout"
	default:
		return "common.operation_failed"
	}
}

func T(key string, args ...Args) string {
	return Render(Current(), key, mergeArgs(args))
}

func Render(locale Locale, key string, args Args) string {
	message, ok := catalogs[locale][key]
	if !ok {
		message = catalogs[locale]["common.unknown_error"]
	}
	return placeholderRE.ReplaceAllStringFunc(message, func(token string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}")
		value, exists := args[name]
		if !exists {
			return token
		}
		return fmt.Sprint(value)
	})
}

func Catalog(locale Locale) map[string]string {
	source := catalogs[locale]
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func ValidateCatalogs() error {
	zh := catalogs[LocaleZhCN]
	en := catalogs[LocaleEnUS]
	var problems []string
	for key, zhMessage := range zh {
		enMessage, ok := en[key]
		if !ok {
			problems = append(problems, "missing en-US key "+key)
			continue
		}
		if !sameStrings(placeholders(zhMessage), placeholders(enMessage)) {
			problems = append(problems, "placeholder mismatch for "+key)
		}
	}
	for key := range en {
		if _, ok := zh[key]; !ok {
			problems = append(problems, "missing zh-CN key "+key)
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func loadCatalogs() (map[Locale]map[string]string, error) {
	result := make(map[Locale]map[string]string, 2)
	for _, locale := range []Locale{LocaleZhCN, LocaleEnUS} {
		content, err := catalogFiles.ReadFile("catalogs/" + string(locale) + ".json")
		if err != nil {
			return nil, err
		}
		var messages map[string]string
		if err := json.Unmarshal(content, &messages); err != nil {
			return nil, fmt.Errorf("decode %s catalog: %w", locale, err)
		}
		result[locale] = messages
	}
	return result, nil
}

func mergeArgs(values []Args) Args {
	result := Args{}
	for _, value := range values {
		for key, item := range value {
			result[key] = item
		}
	}
	return result
}

func placeholders(message string) []string {
	matches := placeholderRE.FindAllStringSubmatch(message, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, match[1])
	}
	sort.Strings(result)
	return result
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
