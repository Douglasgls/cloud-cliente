package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"cloud-client/internal/runtime"
	"cloud-client/pkg/logger"
)

func main() {
	l := logger.New(true)
	mgr := runtime.NewManager(l)

	ctx := context.Background()
	if err := mgr.EnsureDaemonRunning(ctx); err != nil {
		log.Fatalf("EnsureDaemonRunning failed: %v", err)
	}

	fmt.Println("Daemon running.")
	loginServer := "http://201.23.87.235:8080"
	authKey := "hskey-auth-upaRJoe_Y8il-ogdIzkBdowHdEQodH5f3WF4j8iFwHsqyCSCNrB_Xw-41Pq_sj0OhQU-5deukG6Qz"
	hostname := "client-teste-vps"

	fmt.Printf("Running tailscale up to %s...\n", loginServer)
	cmd := exec.Command(mgr.TailscalePath(), "--socket="+mgr.SocketPath(), "up", "--login-server="+loginServer, "--authkey="+authKey, "--hostname="+hostname)
	out, err := cmd.CombinedOutput()
	fmt.Printf("tailscale up output:\n%s\n(err: %v)\n\n", string(out), err)

	time.Sleep(2 * time.Second)

	fmt.Println("Checking tailscale status:")
	statusCmd := exec.Command(mgr.TailscalePath(), "--socket="+mgr.SocketPath(), "status")
	statusOut, statusErr := statusCmd.CombinedOutput()
	fmt.Printf("tailscale status output:\n%s\n(err: %v)\n", string(statusOut), statusErr)
}
