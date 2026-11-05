package utils

import (
	"os/exec"
	"reflect"
	"testing"
)

func TestParseGARRegistry(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantLocation   string
		wantProject    string
		wantRepository string
		wantErr        bool
	}{
		{
			name:           "docker registry format",
			input:          "us-central1-docker.pkg.dev/my-project/my-gar",
			wantLocation:   "us-central1",
			wantProject:    "my-project",
			wantRepository: "my-gar",
			wantErr:        false,
		},
		{
			name:           "generic registry format with multi-region",
			input:          "us-generic.pkg.dev/my-project/my-manifests/",
			wantLocation:   "us",
			wantProject:    "my-project",
			wantRepository: "my-manifests",
			wantErr:        false,
		},
		{
			name:    "invalid host",
			input:   "gcr.io/my-project/my-app",
			wantErr: true,
		},
		{
			name:    "missing repository segment",
			input:   "us-central1-docker.pkg.dev/my-project",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, proj, repo, err := ParseGARRegistry(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseGARRegistry() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if loc != tt.wantLocation || proj != tt.wantProject || repo != tt.wantRepository {
					t.Errorf("ParseGARRegistry() = (%s, %s, %s), want (%s, %s, %s)",
						loc, proj, repo, tt.wantLocation, tt.wantProject, tt.wantRepository)
				}
			}
		})
	}
}

func TestResolveGARConfig(t *testing.T) {
	t.Run("parses from IMAGE_REGISTRY and SERVICE_NAME", func(t *testing.T) {
		envCfg := map[string]interface{}{
			"IMAGE_REGISTRY": "us-central1-docker.pkg.dev/my-project/my-gar",
			"SERVICE_NAME":   "myapp",
		}
		got, err := ResolveGARConfig(envCfg, "", "", "", "", "v1.0.0", "deploy/dev.yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &GARConfig{
			Project:    "my-project",
			Location:   "us-central1",
			Repository: "my-gar",
			Package:    "myapp",
			Version:    "v1.0.0",
			Source:     "deploy/dev.yaml",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("ARTIFACT_REGISTRY and CLI flags override IMAGE_REGISTRY", func(t *testing.T) {
		envCfg := map[string]interface{}{
			"IMAGE_REGISTRY":    "us-central1-docker.pkg.dev/my-project/my-gar",
			"ARTIFACT_REGISTRY": "us-east1-generic.pkg.dev/other-proj/manifests-repo",
			"SERVICE_NAME":      "myapp",
		}
		got, err := ResolveGARConfig(envCfg, "flag-proj", "", "flag-repo", "flag-pkg", "v2.0.0", "deploy/prd.yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &GARConfig{
			Project:    "flag-proj",
			Location:   "us-east1",
			Repository: "flag-repo",
			Package:    "flag-pkg",
			Version:    "v2.0.0",
			Source:     "deploy/prd.yaml",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("returns error when required fields are missing", func(t *testing.T) {
		envCfg := map[string]interface{}{
			"SERVICE_NAME": "myapp",
		}
		if _, err := ResolveGARConfig(envCfg, "", "", "", "", "v1.0.0", "deploy/dev.yaml"); err == nil {
			t.Fatal("expected error for missing GAR project/location/repository, got nil")
		}
	})
}

func TestExecuteGARUpload(t *testing.T) {
	origExec := execCommand
	defer func() { execCommand = origExec }()

	var capturedName string
	var capturedArgs []string
	execCommand = func(name string, arg ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = arg
		return exec.Command("true")
	}

	cfg := &GARConfig{
		Project:    "my-project",
		Location:   "us-central1",
		Repository: "my-gar",
		Package:    "myapp",
		Version:    "2025.07.14.abcdef1",
		Source:     "deploy/dev.yaml",
	}

	if err := ExecuteGARUpload(cfg); err != nil {
		t.Fatalf("ExecuteGARUpload() unexpected error: %v", err)
	}

	if capturedName != "gcloud" {
		t.Errorf("expected binary 'gcloud', got '%s'", capturedName)
	}

	wantArgs := []string{
		"artifacts", "generic", "upload",
		"--project=my-project",
		"--location=us-central1",
		"--repository=my-gar",
		"--package=myapp",
		"--version=2025.07.14.abcdef1",
		"--source=deploy/dev.yaml",
	}
	if !reflect.DeepEqual(capturedArgs, wantArgs) {
		t.Errorf("got args %v, want %v", capturedArgs, wantArgs)
	}
}
