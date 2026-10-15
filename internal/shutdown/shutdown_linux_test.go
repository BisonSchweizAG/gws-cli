//go:build linux

package shutdown

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestIsPowerOff(t *testing.T) {
	tests := map[string]bool{
		"":            true, // systemd < 255 does not provide the type
		"power-off":   true,
		"halt":        true,
		"reboot":      false,
		"soft-reboot": false,
		"kexec":       false,
	}
	for in, want := range tests {
		if got := isPowerOff(in); got != want {
			t.Errorf("isPowerOff(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestShutdownTypeFromMetadata(t *testing.T) {
	tests := []struct {
		name string
		in   dbus.Variant
		want string
	}{
		{
			name: "power-off",
			in: dbus.MakeVariant(map[string]dbus.Variant{
				"preparing": dbus.MakeVariant(true),
				"type":      dbus.MakeVariant("power-off"),
			}),
			want: "power-off",
		},
		{
			name: "reboot",
			in: dbus.MakeVariant(map[string]dbus.Variant{
				"preparing": dbus.MakeVariant(true),
				"type":      dbus.MakeVariant("reboot"),
			}),
			want: "reboot",
		},
		{
			name: "no type",
			in:   dbus.MakeVariant(map[string]dbus.Variant{"preparing": dbus.MakeVariant(false)}),
			want: "",
		},
		{
			name: "wrong type",
			in:   dbus.MakeVariant("foo"),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shutdownTypeFromMetadata(tt.in); got != tt.want {
				t.Errorf("shutdownTypeFromMetadata() = %q, want %q", got, tt.want)
			}
		})
	}
}
