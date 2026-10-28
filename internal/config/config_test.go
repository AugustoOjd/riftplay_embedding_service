package config

import (
	"log/slog"
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr []string // substrings expected in the error
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: Config{Port: 8080, LogLevel: slog.LevelInfo},
		},
		{
			name: "overrides",
			env:  map[string]string{"PORT": "9000", "LOG_LEVEL": "debug"},
			want: Config{Port: 9000, LogLevel: slog.LevelDebug},
		},
		{
			name:    "all errors reported together",
			env:     map[string]string{"PORT": "abc", "LOG_LEVEL": "loud"},
			wantErr: []string{"PORT: not an integer", "LOG_LEVEL"},
		},
		{
			name:    "out of range",
			env:     map[string]string{"PORT": "70000"},
			wantErr: []string{"PORT: 70000 out of range"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Load(env(tt.env))
			if len(tt.wantErr) > 0 {
				if err == nil {
					t.Fatalf("Load() = %+v, want error", got)
				}
				for _, s := range tt.wantErr {
					if !strings.Contains(err.Error(), s) {
						t.Errorf("Load() error = %q, want it to contain %q", err, s)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
