package debugutil

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
)

var sensitiveMarkers = []string{
	"authorization",
	"token",
	"secret",
	"password",
	"passwd",
	"sshkey",
	"ssh_key",
	"apikey",
	"api_key",
	"cloudinit",
	"cloud_init",
	"credential",
}

func Logf(enabled bool, format string, args ...interface{}) {
	if !enabled {
		return
	}

	log.Printf("debug: "+format, args...)
}

func FormatStringMap(values map[string]string) string {
	if len(values) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := values[key]
		if IsSensitiveKey(key) && value != "" {
			value = "<redacted>"
		}
		parts = append(parts, fmt.Sprintf("%s=%q", key, value))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

func RedactURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	query := parsed.Query()
	for key := range query {
		if IsSensitiveKey(key) {
			query.Set(key, "<redacted>")
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func RedactJSONBytes(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}

	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Sprintf("<non-json %d bytes>", len(raw))
	}

	sanitized, err := json.Marshal(Sanitize(value))
	if err != nil {
		return fmt.Sprintf("<json %d bytes>", len(raw))
	}

	return string(sanitized)
}

func Sanitize(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		sanitized := make(map[string]interface{}, len(typed))
		for key, nested := range typed {
			if IsSensitiveKey(key) {
				sanitized[key] = "<redacted>"
				continue
			}
			sanitized[key] = Sanitize(nested)
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(typed))
		for index, nested := range typed {
			sanitized[index] = Sanitize(nested)
		}
		return sanitized
	default:
		return value
	}
}

func IsSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", ""), "_", ""))
	for _, marker := range sensitiveMarkers {
		compacted := strings.ReplaceAll(strings.ReplaceAll(marker, "-", ""), "_", "")
		if strings.Contains(normalized, compacted) {
			return true
		}
	}
	return false
}
