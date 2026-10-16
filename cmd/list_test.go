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
	// Different last started dates (different days for compareDate which truncates to date)
	startTime1 := timestamppb.New(now.Add(-1 * 24 * time.Hour)).AsTime() // yesterday
	startTime2 := timestamppb.New(now.Add(-3 * 24 * time.Hour)).AsTime() // 2 days ago
	// Same last started, different created dates
	createdDate1 := timestamppb.New(now.Add(-1 * 24 * time.Hour)).AsTime() // yesterday
	createdDate2 := timestamppb.New(now.Add(-3 * 24 * time.Hour)).AsTime() // 2 days ago

	workstations := []gcloud.ProjectWorkstation{
		{
			Name:        "ws-z",
			State:       workstationspb.Workstation_STATE_RUNNING,
			LastStarted: &startTime1,
			CreatedDate: &createdDate1,
		},
		{
			Name:        "ws-a",
			State:       workstationspb.Workstation_STATE_STOPPED,
			LastStarted: &startTime2,
			CreatedDate: &createdDate2,
		},
		{
			Name:        "ws-m",
			State:       workstationspb.Workstation_STATE_STARTING,
			LastStarted: &startTime1,
			CreatedDate: &createdDate2,
		},
		{
			Name:        "ws-b",
			State:       workstationspb.Workstation_STATE_RUNNING,
			LastStarted: &startTime1,
			CreatedDate: &createdDate1,
		},
	}

	// Verify default sorting: last-started (newest first) → created (newest first) → name (alphabetical)
	// ws-z and ws-b have same lastStarted (startTime1, newest) and same createdDate (createdDate1, newest)
	// Among them, ws-b comes before ws-z alphabetically
	// ws-m has same lastStarted (startTime1) but older createdDate (createdDate2)
	// ws-a has oldest lastStarted (startTime2)
	expectedOrder := []string{"ws-b", "ws-z", "ws-m", "ws-a"}

	// After sorting, check the order
	sorted := make([]gcloud.ProjectWorkstation, len(workstations))
	copy(sorted, workstations)

	// Simulate the default sort from list.go: last-started → created → name
	for i := range len(sorted) - 1 {
		for j := i + 1; j < len(sorted); j++ {
			cmp := compareDate(sorted[i].LastStarted, sorted[j].LastStarted)
			if cmp == 0 {
				cmp = compareDate(sorted[i].CreatedDate, sorted[j].CreatedDate)
			}
			if cmp == 0 {
				if sorted[i].Name > sorted[j].Name {
					cmp = 1
				} else if sorted[i].Name < sorted[j].Name {
					cmp = -1
				}
			}
			if cmp > 0 {
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

func Test_listRunningFilter(t *testing.T) {
	now := time.Now()
	startTime := timestamppb.New(now.Add(-1 * time.Hour)).AsTime()

	workstations := []gcloud.ProjectWorkstation{
		{
			Name:        "ws-running",
			State:       workstationspb.Workstation_STATE_RUNNING,
			LastStarted: &startTime,
		},
		{
			Name:        "ws-starting",
			State:       workstationspb.Workstation_STATE_STARTING,
			LastStarted: &startTime,
		},
		{
			Name:        "ws-stopping",
			State:       workstationspb.Workstation_STATE_STOPPING,
			LastStarted: &startTime,
		},
		{
			Name:        "ws-stopped",
			State:       workstationspb.Workstation_STATE_STOPPED,
			LastStarted: &startTime,
		},
	}

	// Simulate the running filter from list.go
	filtered := make([]gcloud.ProjectWorkstation, 0, len(workstations))
	for _, ws := range workstations {
		switch ws.State {
		case workstationspb.Workstation_STATE_RUNNING,
			workstationspb.Workstation_STATE_STARTING,
			workstationspb.Workstation_STATE_STOPPING:
			filtered = append(filtered, ws)
		}
	}

	expectedNames := []string{"ws-running", "ws-starting", "ws-stopping"}
	if len(filtered) != len(expectedNames) {
		t.Errorf("running filter returned %d workstations, want %d", len(filtered), len(expectedNames))
	}
	for i, ws := range filtered {
		if ws.Name != expectedNames[i] {
			t.Errorf("position %d: got %s, want %s", i, ws.Name, expectedNames[i])
		}
	}
}

func Test_listFormatState(t *testing.T) {
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
