package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadFromReaderAppliesDefaults(t *testing.T) {
	input := `
http:
  access_token: "token"
  request_per_minute: 0
  retry_after: 0
  retry_attempts: 0
extractor:
  interval: ""
  output_file: ""
  page_limit: 0
`

	cfg, err := LoadFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadFromReader returned error: %v", err)
	}

	if cfg.HTTP.AccessToken != "token" {
		t.Fatalf("AccessToken = %q, want token", cfg.HTTP.AccessToken)
	}
	if cfg.HTTP.RequestPerMinute != DefaultRequestPerMinute {
		t.Fatalf("RequestPerMinute = %d, want %d", cfg.HTTP.RequestPerMinute, DefaultRequestPerMinute)
	}
	if cfg.HTTP.RetryAfter != DefaultRetryAfter {
		t.Fatalf("RetryAfter = %d, want %d", cfg.HTTP.RetryAfter, DefaultRetryAfter)
	}
	if cfg.HTTP.RetryAttempts != DefaultRetryAttempts {
		t.Fatalf("RetryAttempts = %d, want %d", cfg.HTTP.RetryAttempts, DefaultRetryAttempts)
	}
	if cfg.Extractor.Interval != DefaultInterval {
		t.Fatalf("Interval = %s, want %s", cfg.Extractor.Interval, DefaultInterval)
	}
	if cfg.Extractor.OutputFile != DefaultOutputFile {
		t.Fatalf("OutputFile = %q, want %q", cfg.Extractor.OutputFile, DefaultOutputFile)
	}
	if cfg.Extractor.PageLimit != DefaultPageLimit {
		t.Fatalf("PageLimit = %d, want %d", cfg.Extractor.PageLimit, DefaultPageLimit)
	}
}

func TestLoadFromReaderAcceptsOnlySupportedIntervals(t *testing.T) {
	tests := []struct {
		name      string
		interval  string
		want      time.Duration
		wantError bool
	}{
		{name: "five minutes", interval: "5m", want: 5 * time.Minute},
		{name: "thirty seconds", interval: "30s", want: 30 * time.Second},
		{name: "invalid", interval: "1m", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `
http:
  access_token: "token"
extractor:
  interval: "` + tt.interval + `"
`

			cfg, err := LoadFromReader(strings.NewReader(input))
			if tt.wantError {
				if err == nil {
					t.Fatal("LoadFromReader returned nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadFromReader returned error: %v", err)
			}
			if cfg.Extractor.Interval != tt.want {
				t.Fatalf("Interval = %s, want %s", cfg.Extractor.Interval, tt.want)
			}
		})
	}
}

func TestLoadFromReaderRequiresToken(t *testing.T) {
	input := `
http:
  access_token: ""
extractor:
  interval: "5m"
`

	_, err := LoadFromReader(strings.NewReader(input))
	if err == nil {
		t.Fatal("LoadFromReader returned nil error, want missing token error")
	}
}

func TestLoadFromReaderSupportsLegacyTokenKey(t *testing.T) {
	input := `
http:
  asanaRequestToken: "legacy-token"
extractor:
  interval: "30s"
`

	cfg, err := LoadFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadFromReader returned error: %v", err)
	}
	if cfg.HTTP.AccessToken != "legacy-token" {
		t.Fatalf("AccessToken = %q, want legacy-token", cfg.HTTP.AccessToken)
	}
}
