package connect

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/anwendt/dev-connect/internal/config"
	"github.com/anwendt/dev-connect/internal/port"
	"github.com/anwendt/dev-connect/internal/session"
	"github.com/anwendt/dev-connect/internal/sshconfig"
)

const sessionIDBytes = 16

// Options describes a local session preparation request.
type Options struct {
	Config       config.Config
	TargetName   string
	Context      string
	GatewayName  string
	SessionDir   string
	SSHDir       string
	AllocatePort func() (port.Allocation, error)
	Now          func() time.Time
	SessionID    func() string
	Reconnect    bool
}

// Result describes generated local artifacts for a connection attempt.
type Result struct {
	TargetName  string
	Target      config.Target
	GatewayName string
	Gateway     config.Gateway
	LocalHost   string
	LocalPort   int
	SSHFiles    sshconfig.SessionFiles
	SessionDir  string
	State       session.State
}

// Prepare resolves configuration and writes local SSH/session artifacts.
func Prepare(options Options) (Result, error) {
	target, ok := options.Config.Targets[options.TargetName]
	if !ok {
		return Result{}, fmt.Errorf("unknown target %q", options.TargetName)
	}

	gatewayName := options.GatewayName
	if gatewayName == "" {
		gatewayName = target.Gateway
	}
	gateway, ok := options.Config.Gateways[gatewayName]
	if !ok {
		return Result{}, fmt.Errorf("unknown gateway %q", gatewayName)
	}

	hostKey := options.Config.HostKeys[target.HostKeyRef]
	if hostKey == "" {
		return Result{}, fmt.Errorf("missing pinned SSH host key %q", target.HostKeyRef)
	}

	allocatePort := options.AllocatePort
	if allocatePort == nil {
		allocatePort = port.AllocateLoopback
	}
	allocation, err := allocatePort()
	if err != nil {
		return Result{}, fmt.Errorf("allocate local TCP port: %w", err)
	}

	sshDir := options.SSHDir
	if sshDir == "" {
		sshDir = filepath.Join(options.SessionDir, "ssh")
	}
	if sshDir == "" {
		return Result{}, errors.New("session directory or SSH directory is required")
	}

	sshFiles, err := sshconfig.WriteSessionFiles(sshconfig.SessionOptions{
		Dir:       sshDir,
		Alias:     options.TargetName,
		User:      target.User,
		LocalHost: allocation.Host,
		LocalPort: allocation.Port,
		HostKey:   hostKey,
	})
	if err != nil {
		return Result{}, err
	}

	state := session.State{
		SessionID:         sessionID(options.SessionID),
		Target:            options.TargetName,
		Gateway:           gatewayName,
		Namespace:         gateway.Namespace,
		KubernetesContext: kubernetesContext(options.Config, options.Context),
		LocalPort:         allocation.Port,
		SSHConfigPath:     sshFiles.ConfigPath,
		KnownHostsPath:    sshFiles.KnownHostsPath,
		StartedAt:         now(options.Now),
		Reconnect:         options.Reconnect,
	}
	if err := (session.Store{Dir: options.SessionDir}).Save(state); err != nil {
		return Result{}, err
	}

	return Result{
		TargetName:  options.TargetName,
		Target:      target,
		GatewayName: gatewayName,
		Gateway:     gateway,
		LocalHost:   allocation.Host,
		LocalPort:   allocation.Port,
		SSHFiles:    sshFiles,
		SessionDir:  options.SessionDir,
		State:       state,
	}, nil
}

func kubernetesContext(cfg config.Config, contextName string) string {
	if contextName == "" {
		return ""
	}
	context, ok := cfg.Contexts[contextName]
	if !ok {
		return ""
	}
	cluster, ok := cfg.Clusters[context.Cluster]
	if !ok {
		return ""
	}
	return cluster.KubernetesContext
}

func now(fn func() time.Time) time.Time {
	if fn == nil {
		return time.Now().UTC()
	}
	return fn().UTC()
}

func sessionID(fn func() string) string {
	if fn != nil {
		return fn()
	}

	data := make([]byte, sessionIDBytes)
	if _, err := rand.Read(data); err != nil {
		return fmt.Sprintf("session-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(data)
}
