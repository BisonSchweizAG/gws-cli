package cmd

import (
	"testing"
	"time"

	"cloud.google.com/go/workstations/apiv1/workstationspb"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func Test_formatContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      string
		current  string
		contexts map[string]*types.Context
		expected string
	}{
		{"matching context", "my-context", "my-context", nil, "my-context ✅"},
		{"different context", "other-context", "my-context", nil, "other-context"},
		{"empty current context", "my-context", "", nil, "my-context"},
		{"empty both", "", "", nil, ""},
		{"with icon", "my-context", "", map[string]*types.Context{"my-context": {Icon: "🏠"}}, "🏠 my-context"},
		{
			"with icon and matching context",
			"my-context",
			"my-context",
			map[string]*types.Context{"my-context": {Icon: "🏠"}},
			"🏠 my-context ✅",
		},
		{"icon context not in map", "my-context", "", map[string]*types.Context{"other": {Icon: "🏠"}}, "my-context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatContext(tt.ctx, tt.current, tt.contexts); got != tt.expected {
				t.Errorf("formatContext() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func Test_formatState(t *testing.T) {
	tests := []struct {
		state    workstationspb.Workstation_State
		notFound bool
		expected string
	}{
		{workstationspb.Workstation_STATE_RUNNING, false, "🟢 RUNNING"},
		{workstationspb.Workstation_STATE_STOPPED, false, "🔴 STOPPED"},
		{workstationspb.Workstation_STATE_STARTING, false, "🟡 STARTING"},
		{workstationspb.Workstation_STATE_STOPPING, false, "🟡 STOPPING"},
		{workstationspb.Workstation_STATE_UNSPECIFIED, false, "⚪ UNSPECIFIED"},
		{workstationspb.Workstation_STATE_UNSPECIFIED, true, "❌ NOT_FOUND"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := formatState(workstationState{state: tt.state, notFound: tt.notFound}); got != tt.expected {
				t.Errorf("formatState() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func Test_formatUptime(t *testing.T) {
	d1 := time.Second * 90
	d2 := time.Hour * 2
	tests := []struct {
		uptime   *time.Duration
		expected string
	}{
		{nil, ""},
		{&d1, "1m30s"},
		{&d2, "2h"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := formatUptime(tt.uptime); got != tt.expected {
				t.Errorf("formatUptime() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func Test_formatTime(t *testing.T) {
	//nolint:gosmopolitan // Local timezone needed to match formatTime behavior
	d := time.Date(2026, 9, 14, 15, 4, 5, 0, time.Local)
	zero := time.Time{}
	tests := []struct {
		name     string
		t        *time.Time
		expected string
	}{
		{"nil", nil, ""},
		{"zero", &zero, ""},
		{"valid", &d, "15:04:05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTime(tt.t); got != tt.expected {
				t.Errorf("formatTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func Test_formatDateTime(t *testing.T) {
	//nolint:gosmopolitan // Local timezone needed to match formatDateTime behavior
	d := time.Date(2026, 9, 14, 15, 4, 5, 0, time.Local)
	zero := time.Time{}
	tests := []struct {
		name     string
		t        *time.Time
		expected string
	}{
		{"nil", nil, ""},
		{"zero", &zero, ""},
		{"valid", &d, "14.09.2026 15:04:05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDateTime(tt.t); got != tt.expected {
				t.Errorf("formatDateTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}
