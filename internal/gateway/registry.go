//go:build !windows

package gateway

import (
	"context"

	"github.com/bisonschweizag/gws-cli/internal/types"
)

func UpdateDownloadLocation(_ context.Context, _ *types.Config) error {
	return nil
}
