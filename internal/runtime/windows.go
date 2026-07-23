package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (m *Manager) prepareWindows() error {
	if m.logger != nil {
		m.logger.Info("Checking installation...")
	}

	tsPath, tsdPath, found := m.locateWindowsRuntime()
	if !found {
		if m.logger != nil {
			m.logger.Info("Installing Tailscale...")
		}

		if err := m.installWindowsRuntime(); err != nil {
			return fmt.Errorf("failed to install Tailscale on Windows: %w", err)
		}

		tsPath, tsdPath, found = m.locateWindowsRuntime()
		if !found {
			return fmt.Errorf("%w: Tailscale binary not found after installation", ErrRuntimeNotFound)
		}

		if m.logger != nil {
			m.logger.Info("Installation completed.")
		}
	}

	m.tailscalePath = tsPath
	m.tailscaledPath = tsdPath
	m.runtimeDir = filepath.Dir(tsPath)

	home, _ := os.UserHomeDir()
	if home != "" {
		m.stateDir = filepath.Join(home, ".cloud-client", "state")
	} else {
		m.stateDir = filepath.Join(m.runtimeDir, "state")
	}
	_ = os.MkdirAll(m.stateDir, 0755)

	m.socketPath = filepath.Join(m.stateDir, "tailscaled.sock")
	return nil
}

func (m *Manager) locateWindowsRuntime() (string, string, bool) {
	officialDirs := getWindowsCandidateDirs()

	for _, dir := range officialDirs {
		if dir == "" {
			continue
		}
		tsCandidate := filepath.Join(dir, "tailscale.exe")
		tsdCandidate := filepath.Join(dir, "tailscaled.exe")

		if fileExists(tsCandidate) {
			absTS, err := filepath.Abs(tsCandidate)
			if err != nil {
				absTS = tsCandidate
			}

			absTSD := tsdCandidate
			if fileExists(tsdCandidate) {
				if abs, err := filepath.Abs(tsdCandidate); err == nil {
					absTSD = abs
				}
			} else {
				absTSD = absTS // fallback if daemon binary is in same package/service
			}

			return absTS, absTSD, true
		}
	}

	// Last fallback: exec.LookPath
	if lp, err := exec.LookPath("tailscale.exe"); err == nil && lp != "" {
		if abs, err := filepath.Abs(lp); err == nil {
			return abs, abs, true
		}
	}

	return "", "", false
}

func (m *Manager) installWindowsRuntime() error {
	sourceDir := findSourceDir("windows")

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read Windows runtime directory: %w", err)
	}

	var installerPath string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".exe" {
			installerPath = filepath.Join(sourceDir, entry.Name())
			break
		}
	}

	if installerPath == "" {
		return fmt.Errorf("installer executable not found in %s", sourceDir)
	}

	cmd := exec.Command(installerPath, "/quiet", "/norestart")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("silent installer failed: %v (output: %s)", err, string(output))
	}

	return nil
}

func getWindowsCandidateDirs() []string {
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	localAppData := os.Getenv("LocalAppData")

	dirs := []string{
		`C:\Program Files\Tailscale`,
		`C:\Program Files (x86)\Tailscale`,
		`C:\Program Files\Tailscale IPN`,
		`C:\Program Files (x86)\Tailscale IPN`,
	}

	if programFiles != "" {
		dirs = append(dirs, filepath.Join(programFiles, "Tailscale"))
		dirs = append(dirs, filepath.Join(programFiles, "Tailscale IPN"))
	}
	if programFilesX86 != "" {
		dirs = append(dirs, filepath.Join(programFilesX86, "Tailscale"))
		dirs = append(dirs, filepath.Join(programFilesX86, "Tailscale IPN"))
	}
	if localAppData != "" {
		dirs = append(dirs, filepath.Join(localAppData, "Tailscale"))
	}

	return dirs
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
