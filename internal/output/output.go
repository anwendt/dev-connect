package output

import (
	"encoding/json"
	"io"
)

// APIVersion is the current public JSON output API version.
const APIVersion = "v1"

// Response is the common versioned command response shape.
type Response struct {
	APIVersion        string    `json:"apiVersion"`
	Status            string    `json:"status,omitempty"`
	Server            string    `json:"server,omitempty"`
	SessionID         string    `json:"sessionId,omitempty"`
	LocalPort         int       `json:"localPort,omitempty"`
	KubernetesContext string    `json:"kubernetesContext,omitempty"`
	Namespace         string    `json:"namespace,omitempty"`
	Gateway           string    `json:"gateway,omitempty"`
	Reconnect         *bool     `json:"reconnect,omitempty"`
	Uptime            string    `json:"uptime,omitempty"`
	Targets           []Target  `json:"targets,omitempty"`
	Clusters          []Cluster `json:"clusters,omitempty"`
	Gateways          []Gateway `json:"gateways,omitempty"`
	DefaultContext    string    `json:"defaultContext,omitempty"`
	DefaultGateway    string    `json:"defaultGateway,omitempty"`
	Version           string    `json:"version,omitempty"`
	Commit            string    `json:"commit,omitempty"`
	BuildDate         string    `json:"buildDate,omitempty"`
	GoVersion         string    `json:"goVersion,omitempty"`
	OS                string    `json:"os,omitempty"`
	Arch              string    `json:"arch,omitempty"`
}

// Target is a public JSON summary for a configured development target.
type Target struct {
	Name    string `json:"name"`
	Gateway string `json:"gateway"`
	User    string `json:"user,omitempty"`
}

// Cluster is a public JSON summary for a configured Kubernetes cluster.
type Cluster struct {
	Name              string `json:"name"`
	KubernetesContext string `json:"kubernetesContext,omitempty"`
}

// Gateway is a public JSON summary for a configured gateway service.
type Gateway struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
	Port      int    `json:"port"`
}

// WriteJSON writes a versioned command response as JSON.
func WriteJSON(w io.Writer, response Response) error {
	if response.APIVersion == "" {
		response.APIVersion = APIVersion
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(response)
}
