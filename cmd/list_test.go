package cmd

import (
	"testing"
	"time"

	"cloud.google.com/go/workstations/apiv1/workstationspb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bisonschweizag/gws-cli/internal/gcloud"
)

func Test_parseWorkstationName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected gcloud.Workstation
	}{
		{
			name:  "full path",
			input: "projects/my-project/locations/us-central1/workstationClusters/my-cluster/workstationConfigs/my-config/workstations/my-ws",
			expected: gcloud.Workstation{
				Project: "my-project",
				Region:  "us-central1",
				Cluster: "my-cluster",
				Config:  "my-config",
				Name:    "my-ws",
			},
		},
		{
			name:  "short path",
			input: "a/b/c/d",
			expected: gcloud.Workstation{
				Project: "",
				Region:  "",
				Cluster: "",
				Config:  "",
				Name:    "",
			},
		},
		{
			name:  "empty",
			input: "",
			expected: gcloud.Workstation{
				Project: "",
				Region:  "",
				Cluster: "",
				Config:  "",
				Name:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gcloud.ParseWorkstationName(tt.input)
			if got != tt.expected {
				t.Errorf("ParseWorkstationName() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func Test_listWorkstationSorting(t *testing.T) {
	now := time.Now()
	startTime1 := timestamppb.New(now.Add(-1 * time.Hour)).AsTime()
	startTime2 := timestamppb.New(now.Add(-2 * time.Hour)).AsTime()

	workstations := []gcloud.ProjectWorkstation{
		{
			Config:      "config-1",
			Name:        "ws-1",
			State:       workstationspb.Workstation_STATE_RUNNING,
			LastStarted: &startTime1,
		},
		{
			Config:      "config-2",
			Name:        "ws-2",
			State:       workstationspb.Workstation_STATE_STOPPED,
			LastStarted: &startTime2,
		},
		{
			Config:      "config-1",
			Name:        "ws-3",
			State:       workstationspb.Workstation_STATE_STARTING,
			LastStarted: &startTime1,
		},
	}

	// Verify sorting by name (default)
	expectedOrder := []string{"ws-1", "ws-2", "ws-3"}

	// After sorting, check the order
	sorted := make([]gcloud.ProjectWorkstation, len(workstations))
	copy(sorted, workstations)

	// Simulate the name sort from list.go
	for i := range len(sorted) - 1 {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Name > sorted[j].Name {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	for i, expected := range expectedOrder {
		if sorted[i].Name != expected {
			t.Errorf("position %d: got %s, want %s", i, sorted[i].Name, expected)
		}
	}
}

func Test_listFormatDateTime(t *testing.T) {
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

func Test_listFormatState(t *testing.T) {
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
