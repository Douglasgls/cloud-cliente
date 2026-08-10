package runtime

import (
	"fmt"
	"os"
	"path/filepath"
)

func (m *Manager) prepareWindows() error {
	targetDir, err := getWindowsTargetDir()
	if err != nil {
		return fmt.Errorf("could not determine Windows target directory: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create Windows runtime target directory: %w", err)
	}

	tsPath := filepath.Join(targetDir, "tailscale.exe")
	tsdPath := filepath.Join(targetDir, "tailscaled.exe")

	tsExist := fileExists(tsPath)
	tsdExist := fileExists(tsdPath)

	if m.logger != nil {
		tsFoundStr := "missing"
		if tsExist {
			tsFoundStr = "found"
		}
		tsdFoundStr := "missing"
		if tsdExist {
			tsdFoundStr = "found"
		}
		m.logger.Info("Checking runtime binaries...")
		m.logger.Info("tailscale.exe: %s", tsFoundStr)
		m.logger.Info("tailscaled.exe: %s", tsdFoundStr)
	}

	if !tsExist || !tsdExist {
		if err := m.copyWindowsRuntime(targetDir); err != nil {
			return err
		}
	}

	absTS, err := filepath.Abs(tsPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for tailscale.exe: %w", err)
	}

	absTSD, err := filepath.Abs(tsdPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for tailscaled.exe: %w", err)
	}

	stateDir, err := getWindowsStateDir()
	if err != nil {
		return fmt.Errorf("could not determine Windows state directory: %w", err)
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create Windows state directory: %w", err)
	}

	m.runtimeDir = targetDir
	m.stateDir = stateDir
	m.tailscalePath = absTS
	m.tailscaledPath = absTSD
	m.socketPath = `\\.\pipe\cloud-client-tailscaled`

	return nil
}

func (m *Manager) copyWindowsRuntime(targetDir string) error {
	sourceDir := findSourceDir("windows")

	binaries := []string{"tailscale.exe", "tailscaled.exe"}
	for _, bin := range binaries {
		src := filepath.Join(sourceDir, bin)
		dst := filepath.Join(targetDir, bin)

		if !fileExists(src) {
			return fmt.Errorf("runtime source binary missing: %s", src)
		}

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", src, dst, err)
		}
	}

	if m.logger != nil {
		m.logger.Info("Windows runtime binaries copied successfully.")
	}

	return nil
}

func getWindowsTargetDir() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".cloud-client", "runtime"), nil
	}
	return filepath.Abs("./runtime-cache")
}

func getWindowsStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".cloud-client", "state"), nil
	}
	return filepath.Abs("./state-cache")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

