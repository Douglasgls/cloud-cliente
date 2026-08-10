package runtime

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cloud-client/pkg/logger"
)

func TestRuntimeManager_Prepare(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	log := logger.NewWithWriters(out, errOut, true)

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

	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(mgr.TailscalePath(), "tailscale.exe") {
			t.Errorf("expected Windows TailscalePath to end with tailscale.exe, got: %s", mgr.TailscalePath())
		}
		if !strings.HasSuffix(mgr.TailscaledPath(), "tailscaled.exe") {
			t.Errorf("expected Windows TailscaledPath to end with tailscaled.exe, got: %s", mgr.TailscaledPath())
		}
		if mgr.SocketPath() != `\\.\pipe\cloud-client-tailscaled` {
			t.Errorf("expected Windows Named Pipe socket path, got: %s", mgr.SocketPath())
		}
		if !strings.Contains(mgr.RuntimeDir(), ".cloud-client") {
			t.Errorf("expected Windows RuntimeDir to be in .cloud-client, got: %s", mgr.RuntimeDir())
		}
	} else if runtime.GOOS == "linux" {
		if !strings.HasSuffix(mgr.TailscalePath(), "tailscale") {
			t.Errorf("expected Linux TailscalePath to end with tailscale, got: %s", mgr.TailscalePath())
		}
		if !strings.HasSuffix(mgr.TailscaledPath(), "tailscaled") {
			t.Errorf("expected Linux TailscaledPath to end with tailscaled, got: %s", mgr.TailscaledPath())
		}
		if !strings.HasSuffix(mgr.SocketPath(), "tailscaled.sock") {
			t.Errorf("expected Linux socket path to end with tailscaled.sock, got: %s", mgr.SocketPath())
		}
	}

	// Verify runtime directory exists
	if _, err := os.Stat(mgr.RuntimeDir()); err != nil {
		t.Errorf("expected runtime directory %s to exist: %v", mgr.RuntimeDir(), err)
	}

	// Verify state directory exists
	if _, err := os.Stat(mgr.StateDir()); err != nil {
		t.Errorf("expected state directory %s to exist: %v", mgr.StateDir(), err)
	}

	// Calling Prepare() again should be a no-op and succeed
	err = mgr.Prepare()
	if err != nil {
		t.Fatalf("second Prepare() failed: %v", err)
	}
}

func TestRuntimeManager_Directories(t *testing.T) {
	if runtime.GOOS == "windows" {
		targetDir, err := getWindowsTargetDir()
		if err != nil {
			t.Fatalf("getWindowsTargetDir failed: %v", err)
		}
		if !filepath.IsAbs(targetDir) {
			t.Errorf("expected absolute path for getWindowsTargetDir, got: %s", targetDir)
		}

		stateDir, err := getWindowsStateDir()
		if err != nil {
			t.Fatalf("getWindowsStateDir failed: %v", err)
		}
		if !filepath.IsAbs(stateDir) {
			t.Errorf("expected absolute path for getWindowsStateDir, got: %s", stateDir)
		}
	} else if runtime.GOOS == "linux" {
		targetDir, err := getLinuxTargetDir()
		if err != nil {
			t.Fatalf("getLinuxTargetDir failed: %v", err)
		}
		if !filepath.IsAbs(targetDir) {
			t.Errorf("expected absolute path for getLinuxTargetDir, got: %s", targetDir)
		}

		stateDir, err := getLinuxStateDir()
		if err != nil {
			t.Fatalf("getLinuxStateDir failed: %v", err)
		}
		if !filepath.IsAbs(stateDir) {
			t.Errorf("expected absolute path for getLinuxStateDir, got: %s", stateDir)
		}
	}
}
