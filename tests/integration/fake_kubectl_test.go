package integration_test

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFakeKubectlSupportsVersionAndInvocationLog(t *testing.T) {
	fake := buildFakeKubectl(t)
	logPath := filepath.Join(t.TempDir(), "kubectl.log")

	cmd := exec.Command(fake, "version")
	cmd.Env = append(os.Environ(), "DEV_CONNECT_FAKE_KUBECTL_LOG="+logPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fake kubectl version: %v\n%s", err, string(output))
	}
	if !strings.Contains(string(output), "Client Version: v1.30.0") {
		t.Fatalf("unexpected version output:\n%s", string(output))
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read invocation log: %v", err)
	}
	if strings.TrimSpace(string(logData)) != "version" {
		t.Fatalf("invocation log = %q, want version", string(logData))
	}
}

func TestFakeKubectlSupportsRBACDenial(t *testing.T) {
	fake := buildFakeKubectl(t)

	cmd := exec.Command(fake, "auth", "can-i", "create", "pods/portforward")
	cmd.Env = append(os.Environ(), "DEV_CONNECT_FAKE_KUBECTL_CAN_I=no")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("denied can-i succeeded:\n%s", string(output))
	}
	if !strings.Contains(string(output), "no") {
		t.Fatalf("denial output = %q, want no", string(output))
	}
}

func TestFakeKubectlPortForwardBlocksUntilStopped(t *testing.T) {
	fake := buildFakeKubectl(t)

	cmd := exec.Command(fake, "port-forward", "service/dev-connect-gateway-dev01", "55221:22")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("open stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start fake kubectl: %v", err)
	}
	lineCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			lineCh <- scanner.Text()
			return
		}
		lineCh <- ""
	}()

	var line string
	select {
	case line = <-lineCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for port-forward readiness output")
	}
	if cmd.ProcessState != nil {
		t.Fatal("port-forward process exited before explicit stop")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill fake kubectl: %v", err)
	}
	err = cmd.Wait()
	if err == nil {
		t.Fatal("killed port-forward process exited successfully")
	}
	if !strings.Contains(line, "Forwarding from 127.0.0.1:55221 -> 22") {
		t.Fatalf("port-forward output = %q", line)
	}
}

func buildFakeKubectl(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "kubectl")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./tests/fakes/kubectl")
	cmd.Dir = projectRoot(t)
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(projectRoot(t), ".cache", "go-build"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build fake kubectl: %v\n%s", err, string(output))
	}
	return binary
}
