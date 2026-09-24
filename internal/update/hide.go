//go:build !windows

package update

func hideFile(string) error { return nil }
