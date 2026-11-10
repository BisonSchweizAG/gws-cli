package gateway

import (
	"slices"
	"strings"
	"testing"

	"github.com/bisonschweizag/gws-cli/internal/log"
	"github.com/bisonschweizag/gws-cli/internal/types"
)

func TestRunLogsSkippedTasks(t *testing.T) {
	tests := []struct {
		name string
		cfg  *types.Config
		want []string
	}{
		{
			name: "nil config",
			want: []string{
				"JetBrains Gateway downloadDestination is not configured; skipping",
				"Checking JetBrains Gateway proxy settings",
				`No JetBrains Gateway config directories found in "{base}"`,
				"JetBrains Gateway tasks finished",
			},
		},
		{
			name: "no gateway config",
			cfg:  &types.Config{},
			want: []string{
				"JetBrains Gateway downloadDestination is not configured; skipping",
				"Checking JetBrains Gateway proxy settings",
				`No JetBrains Gateway config directories found in "{base}"`,
				"JetBrains Gateway tasks finished",
			},
		},
		{
			name: "proxy check skipped",
			cfg:  &types.Config{JetbrainsGateway: &types.GatewayConfig{SkipNoProxyCheck: true}},
			want: []string{
				"JetBrains Gateway downloadDestination is not configured; skipping",
				"JetBrains Gateway skipNoProxyCheck is enabled; skipping proxy settings",
				"JetBrains Gateway tasks finished",
			},
		},
	}

	previous := log.GetLogger()
	t.Cleanup(func() { log.SetLogger(previous) })

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := isolateUserConfigDirForTest(t)
			want := make([]string, len(tt.want))
			for i, line := range tt.want {
				want[i] = strings.ReplaceAll(line, "{base}", baseDir)
			}

			var got []string
			log.SetLogger(func(s string) { got = append(got, s) })

			if err := Run(t.Context(), tt.cfg); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !slices.Equal(got, want) {
				t.Fatalf("Run() logs = %q, want %q", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
		})
	}
}
