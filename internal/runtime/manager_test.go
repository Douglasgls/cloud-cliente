package runtime

import (
	"bytes"
	"testing"

	"cloud-client/pkg/logger"
)

func TestRuntimeManager_Prepare(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	log := logger.NewWithWriters(out, errOut, false)

	mgr := NewManager(log)
	if mgr.IsPrepared() {
		t.Fatal("expected IsPrepared() to be false initially")
	}

	err := mgr.Prepare()
	if err != nil {
		t.Fatalf("Prepare() failed: %v", err)
	}

	if !mgr.IsPrepared() {
		t.Fatal("expected IsPrepared() to be true after Prepare()")
	}

	if mgr.TailscalePath() == "" {
		t.Error("expected TailscalePath() to be non-empty")
	}

	if mgr.TailscaledPath() == "" {
		t.Error("expected TailscaledPath() to be non-empty")
	}

	// Calling Prepare() again should be a no-op and succeed
	err = mgr.Prepare()
	if err != nil {
		t.Fatalf("second Prepare() failed: %v", err)
	}
}
