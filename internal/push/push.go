package push

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/runparcel/runparcel/internal/utils"
)

const (
	defaultDeployDir  = "deploy"
	defaultValuesPath = "values.yaml"
)

// Options holds the configuration options for the push command.
type Options struct {
	ValuesPath string
	DeployDir  string
	Tag        string
	Env        string
	Project    string
	Location   string
	Repository string
	Package    string
}

// Cmd returns the push cobra command.
func Cmd() *cobra.Command {
	var opts Options

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push generated deployment YAML files to Google Artifact Registry",
		Long:  `Push generated Cloud Run deployment YAML files from the deploy/ directory to Google Artifact Registry (GAR) using gcloud artifacts generic upload.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return Push(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ValuesPath, "values", "v", "", "Path to the values file (default: values.yaml)")
	cmd.Flags().StringVarP(&opts.DeployDir, "deploy-dir", "d", "", "Directory containing generated YAML files (default: deploy)")
	cmd.Flags().StringVarP(&opts.Tag, "tag", "", "", "Specify an artifact version tag (auto-generated if omitted)")
	cmd.Flags().StringVarP(&opts.Env, "env", "e", "", "Push only a specific environment (default: all environments)")
	cmd.Flags().StringVarP(&opts.Project, "project", "", "", "Override GCP project ID for Google Artifact Registry")
	cmd.Flags().StringVarP(&opts.Location, "location", "", "", "Override GCP location/region for Google Artifact Registry")
	cmd.Flags().StringVarP(&opts.Repository, "repository", "", "", "Override Google Artifact Registry repository name")
	cmd.Flags().StringVarP(&opts.Package, "package", "", "", "Override Google Artifact Registry package name (defaults to SERVICE_NAME)")

	return cmd
}

// Push reads values.yaml, resolves GAR settings for each environment, and uploads the generated deployment YAMLs.
func Push(opts Options) error {
	if opts.ValuesPath == "" {
		opts.ValuesPath = defaultValuesPath
	}
	if opts.DeployDir == "" {
		opts.DeployDir = defaultDeployDir
	}

	values, err := readValuesFile(opts.ValuesPath)
	if err != nil {
		return fmt.Errorf("error reading values.yaml: %v", err)
	}

	environments, ok := values["environments"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("top-level 'environments' key not found or is invalid in values.yaml")
	}

	resolvedTag, err := utils.GetResolvedTag(opts.Tag)
	if err != nil {
		return err
	}

	commonConfig := make(map[string]interface{})
	for key, val := range values {
		if key != "environments" {
			commonConfig[key] = val
		}
	}

	envNames := make([]string, 0, len(environments))
	if opts.Env != "" {
		if _, exists := environments[opts.Env]; !exists {
			return fmt.Errorf("environment '%s' not found in values.yaml", opts.Env)
		}
		envNames = append(envNames, opts.Env)
	} else {
		for envName := range environments {
			envNames = append(envNames, envName)
		}
		sort.Strings(envNames)
	}

	for _, envName := range envNames {
		envConfigGeneric := environments[envName]
		envConfig, ok := envConfigGeneric.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid configuration for environment %s: expected a map", envName)
		}

		finalEnvConfig := make(map[string]interface{})
		for k, v := range commonConfig {
			finalEnvConfig[k] = v
		}
		for k, v := range envConfig {
			finalEnvConfig[k] = v
		}

		yamlFile := filepath.Join(opts.DeployDir, fmt.Sprintf("%s.yaml", envName))
		if _, err := os.Stat(yamlFile); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("deployment YAML '%s' not found for environment '%s'. Run 'runparcel generate' first", yamlFile, envName)
			}
			return fmt.Errorf("error checking file '%s': %v", yamlFile, err)
		}

		garCfg, err := utils.ResolveGARConfig(
			finalEnvConfig,
			opts.Project,
			opts.Location,
			opts.Repository,
			opts.Package,
			resolvedTag,
			yamlFile,
		)
		if err != nil {
			return fmt.Errorf("error resolving GAR configuration for environment '%s': %v", envName, err)
		}

		if err := utils.ExecuteGARUpload(garCfg); err != nil {
			return fmt.Errorf("error pushing YAML for environment '%s': %v", envName, err)
		}
	}

	return nil
}

func readValuesFile(valuesPath string) (map[string]interface{}, error) {
	valuesFile, err := os.ReadFile(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("error reading values.yaml at %s: %v", valuesPath, err)
	}

	var values map[string]interface{}
	if err := yaml.Unmarshal(valuesFile, &values); err != nil {
		return nil, fmt.Errorf("error unmarshalling values.yaml: %v", err)
	}

	return values, nil
}
