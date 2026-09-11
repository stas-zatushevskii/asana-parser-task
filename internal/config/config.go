package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	transporthttp "asana/internal/transport/http"
)

const (
	DefaultRequestPerMinute = 150
	DefaultRetryAfter       = 30
	DefaultRetryAttempts    = 3
	DefaultInterval         = 5 * time.Minute
	DefaultOutputFile       = "output/asana_objects.json"
	DefaultPageLimit        = 100
)

type Config struct {
	HTTP      transporthttp.Config
	Extractor ExtractorConfig
}

type ExtractorConfig struct {
	Interval   time.Duration
	OutputFile string
	PageLimit  int
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	config, err := LoadFromReader(file)
	if err != nil {
		return Config{}, fmt.Errorf("load config %q: %w", path, err)
	}

	return config, nil
}

func LoadFromReader(reader io.Reader) (Config, error) {
	rawConfig, err := parseYAML(reader)
	if err != nil {
		return Config{}, err
	}

	config := Config{
		HTTP: transporthttp.Config{
			AccessToken:      value(rawConfig, "http", "access_token"),
			RequestPerMinute: intValue(rawConfig, "http", "request_per_minute"),
			RetryAfter:       intValue(rawConfig, "http", "retry_after"),
			RetryAttempts:    intValue(rawConfig, "http", "retry_attempts"),
		},
		Extractor: ExtractorConfig{
			OutputFile: value(rawConfig, "extractor", "output_file"),
			PageLimit:  intValue(rawConfig, "extractor", "page_limit"),
		},
	}

	if config.HTTP.AccessToken == "" {
		config.HTTP.AccessToken = value(rawConfig, "http", "asanaRequestToken")
	}

	intervalValue := value(rawConfig, "extractor", "interval")
	interval, err := normalizeInterval(intervalValue)
	if err != nil {
		return Config{}, err
	}
	config.Extractor.Interval = interval

	if err := config.Normalize(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (config *Config) Normalize() error {
	config.HTTP.AccessToken = strings.TrimSpace(config.HTTP.AccessToken)
	if config.HTTP.AccessToken == "" {
		return fmt.Errorf("http.access_token is required")
	}

	if config.HTTP.RequestPerMinute <= 0 {
		config.HTTP.RequestPerMinute = DefaultRequestPerMinute
	}
	if config.HTTP.RetryAfter <= 0 {
		config.HTTP.RetryAfter = DefaultRetryAfter
	}
	if config.HTTP.RetryAttempts <= 0 {
		config.HTTP.RetryAttempts = DefaultRetryAttempts
	}

	if config.Extractor.Interval == 0 {
		config.Extractor.Interval = DefaultInterval
	}
	if config.Extractor.Interval != 5*time.Minute && config.Extractor.Interval != 30*time.Second {
		return fmt.Errorf("extractor.interval must be either 5m or 30s")
	}

	config.Extractor.OutputFile = strings.TrimSpace(config.Extractor.OutputFile)
	if config.Extractor.OutputFile == "" {
		config.Extractor.OutputFile = DefaultOutputFile
	}

	if config.Extractor.PageLimit <= 0 {
		config.Extractor.PageLimit = DefaultPageLimit
	}
	if config.Extractor.PageLimit > DefaultPageLimit {
		config.Extractor.PageLimit = DefaultPageLimit
	}

	return nil
}

func normalizeInterval(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultInterval, nil
	}

	switch raw {
	case "5m", "5min", "5 minutes", "5 minute":
		return 5 * time.Minute, nil
	case "30s", "30sec", "30 seconds", "30 second":
		return 30 * time.Second, nil
	default:
		return 0, fmt.Errorf("extractor.interval must be either 5m or 30s")
	}
}

func parseYAML(reader io.Reader) (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)
	scanner := bufio.NewScanner(reader)
	section := ""
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := stripComment(scanner.Text())
		if strings.TrimSpace(line) == "" {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, ":") {
			return nil, fmt.Errorf("line %d: expected key/value pair", lineNumber)
		}

		if !strings.HasPrefix(line, " ") && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			result[section] = make(map[string]string)
			continue
		}
		if section == "" {
			return nil, fmt.Errorf("line %d: key outside section", lineNumber)
		}

		key, rawValue, ok := strings.Cut(trimmed, ":")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key/value pair", lineNumber)
		}
		result[section][strings.TrimSpace(key)] = unquote(strings.TrimSpace(rawValue))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	return result, nil
}

func stripComment(line string) string {
	inSingleQuote := false
	inDoubleQuote := false
	for i, ch := range line {
		switch ch {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		case '#':
			if !inSingleQuote && !inDoubleQuote {
				return line[:i]
			}
		}
	}

	return line
}

func unquote(value string) string {
	if len(value) < 2 {
		return value
	}
	if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
		(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
		return value[1 : len(value)-1]
	}

	return value
}

func value(values map[string]map[string]string, section string, key string) string {
	if values[section] == nil {
		return ""
	}

	return strings.TrimSpace(values[section][key])
}

func intValue(values map[string]map[string]string, section string, key string) int {
	raw := value(values, section, key)
	if raw == "" {
		return 0
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}

	return parsed
}
