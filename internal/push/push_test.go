package push

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushMissingDeployFile(t *testing.T) {
	tmpDir := t.TempDir()
	valuesPath := filepath.Join(tmpDir, "values.yaml")
	deployDir := filepath.Join(tmpDir, "deploy")

	valuesContent := `
IMAGE_REGISTRY: us-central1-docker.pkg.dev/my-project/my-gar
SERVICE_NAME: myapp
environments:
  dev:
    REGION: us-central1
`
	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("failed to write temp values.yaml: %v", err)
	}

	err := Push(Options{
		ValuesPath: valuesPath,
		DeployDir:  deployDir,
		Tag:        "v1.0.0",
	})
	if err == nil {
		t.Fatal("expected error when deploy/dev.yaml is missing, got nil")
	}
	if !strings.Contains(err.Error(), "Run 'runparcel generate' first") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPushUnknownEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	valuesPath := filepath.Join(tmpDir, "values.yaml")

	valuesContent := `
IMAGE_REGISTRY: us-central1-docker.pkg.dev/my-project/my-gar
SERVICE_NAME: myapp
environments:
  dev:
    REGION: us-central1
`
	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("failed to write temp values.yaml: %v", err)
	}

	err := Push(Options{
		ValuesPath: valuesPath,
		DeployDir:  tmpDir,
		Tag:        "v1.0.0",
		Env:        "staging",
	})
	if err == nil {
		t.Fatal("expected error for unknown environment, got nil")
	}
	if !strings.Contains(err.Error(), "environment 'staging' not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}
