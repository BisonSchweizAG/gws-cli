package cmd

import (
	"testing"
	"time"

	"cloud.google.com/go/workstations/apiv1/workstationspb"
)

func Test_formatState(t *testing.T) {
	tests := []struct {
		state    workstationspb.Workstation_State
		expected string
	}{
		{workstationspb.Workstation_STATE_RUNNING, "🟢 RUNNING"},
		{workstationspb.Workstation_STATE_STOPPED, "🔴 STOPPED"},
		{workstationspb.Workstation_STATE_STARTING, "🟡 STARTING"},
		{workstationspb.Workstation_STATE_STOPPING, "🟡 STOPPING"},
		{workstationspb.Workstation_STATE_UNSPECIFIED, "⚪ UNSPECIFIED"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := formatState(tt.state); got != tt.expected {
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
		{&d2, "2h0m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := formatUptime(tt.uptime); got != tt.expected {
				t.Errorf("formatUptime() = %v, want %v", got, tt.expected)
			}
		})
	}
}
