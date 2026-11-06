package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GARConfig holds the resolved configuration needed to upload an artifact to Google Artifact Registry.
type GARConfig struct {
	Project    string
	Location   string
	Repository string
	Package    string
	Version    string
	Source     string
}

// execCommand is a package-level variable to allow mocking in tests.
var execCommand = exec.Command

// ParseGARRegistry parses a Google Artifact Registry URI of the form:
// <location>-<format>.pkg.dev/<project>/<repository> (e.g. us-central1-docker.pkg.dev/my-project/my-gar)
// and returns the location, project, and repository.
func ParseGARRegistry(registry string) (location, project, repository string, err error) {
	trimmed := strings.TrimSpace(registry)
	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	trimmed = strings.TrimSuffix(trimmed, "/")

	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("invalid Artifact Registry format '%s': expected <location>-<format>.pkg.dev/<project>/<repository>", registry)
	}

	host := parts[0]
	if !strings.HasSuffix(host, ".pkg.dev") {
		return "", "", "", fmt.Errorf("invalid Artifact Registry host '%s': expected '*.pkg.dev'", host)
	}

	hostPrefix := strings.TrimSuffix(host, ".pkg.dev")
	lastDash := strings.LastIndex(hostPrefix, "-")
	if lastDash <= 0 {
		return "", "", "", fmt.Errorf("unable to extract location from Artifact Registry host '%s'", host)
	}

	location = hostPrefix[:lastDash]
	project = parts[1]
	repository = parts[2]
	return location, project, repository, nil
}

// ResolveGARConfig resolves the GAR upload parameters by combining values from values.yaml
// (parsed from ARTIFACT_REGISTRY / GAR_REPO / IMAGE_REGISTRY and explicit GAR_* keys) with CLI flag overrides.
func ResolveGARConfig(
	envConfig map[string]interface{},
	flagProject, flagLocation, flagRepository, flagPackage string,
	resolvedTag, sourceFile string,
) (*GARConfig, error) {
	cfg := &GARConfig{
		Version: resolvedTag,
		Source:  sourceFile,
	}

	// 1. Check registry URL keys in order of precedence: ARTIFACT_REGISTRY, GAR_REPO, IMAGE_REGISTRY
	for _, key := range []string{"ARTIFACT_REGISTRY", "GAR_REPO", "IMAGE_REGISTRY"} {
		if raw, ok := getStringValue(envConfig, key); ok && strings.Contains(raw, ".pkg.dev/") {
			loc, proj, repo, err := ParseGARRegistry(raw)
			if err == nil {
				cfg.Location = loc
				cfg.Project = proj
				cfg.Repository = repo
				break
			}
		}
	}

	// 2. Apply explicit keys from values.yaml if present
	if proj, ok := getStringValue(envConfig, "GAR_PROJECT"); ok {
		cfg.Project = proj
	}
	if loc, ok := getStringValue(envConfig, "GAR_LOCATION"); ok {
		cfg.Location = loc
	} else if cfg.Location == "" {
		if region, ok := getStringValue(envConfig, "REGION"); ok {
			cfg.Location = region
		}
	}
	if repo, ok := getStringValue(envConfig, "GAR_REPOSITORY"); ok {
		cfg.Repository = repo
	} else if repo, ok := getStringValue(envConfig, "GAR_REPO"); ok && !strings.Contains(repo, "/") {
		cfg.Repository = repo
	}
	if pkg, ok := getStringValue(envConfig, "GAR_PACKAGE"); ok {
		cfg.Package = pkg
	} else if svc, ok := getStringValue(envConfig, "SERVICE_NAME"); ok {
		cfg.Package = svc
	}

	// 3. Apply CLI flag overrides
	if flagProject != "" {
		cfg.Project = flagProject
	}
	if flagLocation != "" {
		cfg.Location = flagLocation
	}
	if flagRepository != "" {
		cfg.Repository = flagRepository
	}
	if flagPackage != "" {
		cfg.Package = flagPackage
	}

	// 4. Validate required fields
	if cfg.Project == "" {
		return nil, fmt.Errorf("GAR project could not be determined; set 'ARTIFACT_REGISTRY', 'IMAGE_REGISTRY', or 'GAR_PROJECT' in values.yaml, or pass --project")
	}
	if cfg.Location == "" {
		return nil, fmt.Errorf("GAR location could not be determined; set 'ARTIFACT_REGISTRY', 'IMAGE_REGISTRY', 'GAR_LOCATION', or 'REGION' in values.yaml, or pass --location")
	}
	if cfg.Repository == "" {
		return nil, fmt.Errorf("GAR repository could not be determined; set 'ARTIFACT_REGISTRY', 'GAR_REPO', 'GAR_REPOSITORY', or 'IMAGE_REGISTRY' in values.yaml, or pass --repository")
	}
	if cfg.Package == "" {
		return nil, fmt.Errorf("GAR package name could not be determined; set 'SERVICE_NAME' or 'GAR_PACKAGE' in values.yaml, or pass --package")
	}

	return cfg, nil
}

// ExecuteGARUpload uploads a deployment YAML file to Google Artifact Registry using gcloud.
func ExecuteGARUpload(cfg *GARConfig) error {
	fmt.Printf(
		"Info: Uploading '%s' to Google Artifact Registry (project=%s, location=%s, repository=%s, package=%s, version=%s)...\n",
		cfg.Source, cfg.Project, cfg.Location, cfg.Repository, cfg.Package, cfg.Version,
	)

	cmd := execCommand(
		"gcloud", "artifacts", "generic", "upload",
		fmt.Sprintf("--project=%s", cfg.Project),
		fmt.Sprintf("--location=%s", cfg.Location),
		fmt.Sprintf("--repository=%s", cfg.Repository),
		fmt.Sprintf("--package=%s", cfg.Package),
		fmt.Sprintf("--version=%s", cfg.Version),
		fmt.Sprintf("--source=%s", cfg.Source),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"failed to upload '%s' to Google Artifact Registry (%s/%s/%s:%s). Ensure 'gcloud' is installed, authenticated, and the generic repository exists: %v",
			cfg.Source, cfg.Location, cfg.Project, cfg.Repository, cfg.Version, err,
		)
	}

	fmt.Printf("Info: Successfully uploaded '%s' to Google Artifact Registry.\n", cfg.Source)
	return nil
}

func getStringValue(m map[string]interface{}, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	val, ok := m[key].(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}
