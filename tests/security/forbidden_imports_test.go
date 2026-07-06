package security_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var forbiddenImports = []string{
	`"k8s.io/client-go`,
	`"sigs.k8s.io/controller-runtime`,
	`"github.com/rancher/`,
	`"github.com/rancher-sandbox/`,
	`"net/http"`,
	`"net/http/httputil"`,
	`"golang.org/x/net/websocket"`,
	`"nhooyr.io/websocket"`,
	`"github.com/gorilla/websocket"`,
}

func TestClientDoesNotImportForbiddenKubernetesRancherOrDirectHTTPClients(t *testing.T) {
	root := filepath.Clean("../..")
	scanRoots := []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, "internal"),
	}

	for _, scanRoot := range scanRoots {
		err := filepath.WalkDir(scanRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)
			for _, forbidden := range forbiddenImports {
				if strings.Contains(content, forbidden) {
					t.Errorf("%s imports forbidden dependency pattern %s", path, forbidden)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", scanRoot, err)
		}
	}
}
