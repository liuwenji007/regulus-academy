package cliruntime

import (
	"testing"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestRemoteProgressWins(t *testing.T) {
	localAt := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	local := &storage.UserProgress{UpdatedAt: localAt}

	tests := []struct {
		name      string
		local     *storage.UserProgress
		remoteAt  string
		wantMerge bool
	}{
		{name: "no local", local: nil, remoteAt: "", wantMerge: true},
		{name: "remote newer", local: local, remoteAt: "2026-06-17T13:00:00Z", wantMerge: true},
		{name: "remote older", local: local, remoteAt: "2026-06-17T11:00:00Z", wantMerge: false},
		{name: "remote same time", local: local, remoteAt: "2026-06-17T12:00:00Z", wantMerge: false},
		{name: "remote missing timestamp", local: local, remoteAt: "", wantMerge: false},
		{name: "remote malformed timestamp", local: local, remoteAt: "not-a-date", wantMerge: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := remoteProgressWins(tc.local, tc.remoteAt)
			if got != tc.wantMerge {
				t.Fatalf("remoteProgressWins() = %v, want %v", got, tc.wantMerge)
			}
		})
	}
}
