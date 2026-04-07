package citest

import "testing"

func BoolPtr(v bool) *bool {
	return &v
}

type EnabledCase[Ctx any, Cfg any] struct {
	Name     string
	Context  Ctx
	HasToken bool
	Config   Cfg
	Expected bool
}

func RunEnabledCases[Ctx any, Cfg any](
	t *testing.T,
	cases []EnabledCase[Ctx, Cfg],
	isEnabled func(t *testing.T, ctx Ctx, cfg Cfg, hasToken bool) bool,
) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if got := isEnabled(t, tc.Context, tc.Config, tc.HasToken); got != tc.Expected {
				t.Fatalf("IsEnabled() = %v, expected %v", got, tc.Expected)
			}
		})
	}
}

func AssertCreateOnly(tb testing.TB, createdBody, updatedBody string) {
	tb.Helper()
	if createdBody == "" {
		tb.Fatal("expected create to be called")
	}
	if updatedBody != "" {
		tb.Fatal("did not expect update to be called")
	}
}

func AssertUpdateOnly(tb testing.TB, createdBody, updatedBody string, gotID, wantID int64) {
	tb.Helper()
	if createdBody != "" {
		tb.Fatal("did not expect create to be called")
	}
	if updatedBody == "" {
		tb.Fatal("expected update to be called")
	}
	if gotID != wantID {
		tb.Fatalf("expected update id %d, got %d", wantID, gotID)
	}
}
