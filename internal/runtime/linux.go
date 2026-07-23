package runtime

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (m *Manager) prepareLinux() error {
	targetDir, err := getLinuxTargetDir()
	if err != nil {
		return fmt.Errorf("could not determine target directory: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create runtime target directory: %w", err)
	}

	tsPath := filepath.Join(targetDir, "tailscale")
	tsdPath := filepath.Join(targetDir, "tailscaled")

	tsExist := isExecutable(tsPath)
	tsdExist := isExecutable(tsdPath)

	if !tsExist || !tsdExist {
		if err := m.copyLinuxRuntime(targetDir); err != nil {
			return err
		}
	}

	absTS, err := filepath.Abs(tsPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for tailscale: %w", err)
	}

	absTSD, err := filepath.Abs(tsdPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for tailscaled: %w", err)
	}

	stateDir, err := getLinuxStateDir()
	if err != nil {
		return fmt.Errorf("could not determine state directory: %w", err)
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	m.runtimeDir = targetDir
	m.stateDir = stateDir
	m.tailscalePath = absTS
	m.tailscaledPath = absTSD
	m.socketPath = filepath.Join(stateDir, "tailscaled.sock")
	return nil
}

func (m *Manager) copyLinuxRuntime(targetDir string) error {
	sourceDir := findSourceDir("linux")

	binaries := []string{"tailscale", "tailscaled"}
	for _, bin := range binaries {
		src := filepath.Join(sourceDir, bin)
		dst := filepath.Join(targetDir, bin)

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", src, dst, err)
		}

		if err := os.Chmod(dst, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions on %s: %w", dst, err)
		}
	}

	if m.logger != nil {
		m.logger.Info("Runtime copied.")
	}

	return nil
}

func getLinuxTargetDir() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".cloud-client", "runtime"), nil
	}
	return filepath.Abs("./runtime-cache")
}

func getLinuxStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".cloud-client", "state"), nil
	}
	return filepath.Abs("./state-cache")
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	// Check if file is executable by user (0100 bit)
	return info.Mode()&0111 != 0
}

func findSourceDir(osSubdir string) string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	dir := cwd
	for i := 0; i < 5; i++ {
		candidate1 := filepath.Join(dir, "runtime", osSubdir)
		if info, err := os.Stat(candidate1); err == nil && info.IsDir() {
			return candidate1
		}
		candidate2 := filepath.Join(dir, "assets", "runtime", osSubdir)
		if info, err := os.Stat(candidate2); err == nil && info.IsDir() {
			return candidate2
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("runtime", osSubdir)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
