package log

import (
	"testing"
)

func TestSetLevel(t *testing.T) {
	SetLevel(DebugLevel)
	if !IsDebug() {
		t.Error("expected IsDebug() true after SetLevel(DebugLevel)")
	}

	SetLevel(InfoLevel)
	if IsDebug() {
		t.Error("expected IsDebug() false after SetLevel(InfoLevel)")
	}
}

func TestSetLevelFromString(t *testing.T) {
	tests := []struct {
		level   string
		wantErr bool
		isDebug bool
	}{
		{"debug", false, true},
		{"info", false, false},
		{"warn", false, false},
		{"error", false, false},
		{"invalid", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			err := SetLevelFromString(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetLevelFromString(%q) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}
			if !tt.wantErr && IsDebug() != tt.isDebug {
				t.Errorf("IsDebug() = %v after SetLevelFromString(%q), want %v", IsDebug(), tt.level, tt.isDebug)
			}
		})
	}
}

func TestInit(t *testing.T) {
	Init()
	if IsDebug() {
		t.Error("expected IsDebug() false after Init()")
	}
}

func TestLogFunctions(t *testing.T) {
	// Verify log functions don't panic.
	Init()
	SetLevel(DebugLevel)

	Debug("debug msg")
	Debugf("debug %s", "formatted")
	Info("info msg")
	Infof("info %s", "formatted")
	Warn("warn msg")
	Warnf("warn %s", "formatted")
	Error("error msg")
	Errorf("error %s", "formatted")
	// Skip Fatal/Fatalf — they call os.Exit

	entry := WithField("key", "value")
	if entry == nil {
		t.Error("WithField should return non-nil entry")
	}

	entry = WithError(nil)
	if entry == nil {
		t.Error("WithError should return non-nil entry")
	}
}

func TestPadding(_ *testing.T) {
	Init()
	// Verify padding functions don't panic
	IncreasePadding()
	DecreasePadding()
	ResetPadding()
}

func TestIsDebug(t *testing.T) {
	SetLevel(DebugLevel)
	if !IsDebug() {
		t.Error("expected true for DebugLevel")
	}

	SetLevel(WarnLevel)
	if IsDebug() {
		t.Error("expected false for WarnLevel")
	}
}
