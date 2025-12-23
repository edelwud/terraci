package gitlab

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

func TestMRService_IsEnabled(t *testing.T) {
	t.Run("not in MR", func(t *testing.T) {
		svc := &MRService{
			context: &MRContext{InMR: false},
			client:  NewClient("", "token"),
		}
		if svc.IsEnabled() {
			t.Error("expected IsEnabled to be false when not in MR")
		}
	})

	t.Run("in MR without token", func(t *testing.T) {
		svc := &MRService{
			context: &MRContext{InMR: true},
			client:  NewClient("", ""),
		}
		if svc.IsEnabled() {
			t.Error("expected IsEnabled to be false without token")
		}
	})

	t.Run("in MR with token, default config", func(t *testing.T) {
		svc := &MRService{
			context: &MRContext{InMR: true},
			client:  NewClient("", "token"),
			config:  nil,
		}
		if !svc.IsEnabled() {
			t.Error("expected IsEnabled to be true by default")
		}
	})

	t.Run("explicitly disabled", func(t *testing.T) {
		enabled := false
		svc := &MRService{
			context: &MRContext{InMR: true},
			client:  NewClient("", "token"),
			config: &config.MRConfig{
				Comment: &config.MRCommentConfig{
					Enabled: &enabled,
				},
			},
		}
		if svc.IsEnabled() {
			t.Error("expected IsEnabled to be false when explicitly disabled")
		}
	})
}
