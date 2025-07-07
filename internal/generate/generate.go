package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	deployDir = "deploy" // Directory to store generated YAML files
)

// Cmd returns the generate command
func Cmd() *cobra.Command {
	var templatePath, valuesPath string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate deployment YAML files",
		Long:  "Generate deployment YAML files for Cloud Run environments.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set default paths if not provided
			if templatePath == "" {
				templatePath = filepath.Join("cloudrun", "run.yaml.tmpl")
			}
			if valuesPath == "" {
				valuesPath = "values.yaml"
			}

			return Generate(templatePath, valuesPath)
		},
	}

	// Add flags for template and values paths
	cmd.Flags().StringVarP(&templatePath, "template", "t", "", "Path to the template file (default: cloudrun/run.yaml.tmpl)")
	cmd.Flags().StringVarP(&valuesPath, "values", "v", "", "Path to the values file (default: values.yaml)")

	return cmd
}

// Generate reads the template and values, then generates YAML files
func Generate(templatePath, valuesPath string) error {
	// Read and parse values.yaml
	values, err := readValuesFile(valuesPath)
	if err != nil {
		return fmt.Errorf("error reading values.yaml: %v", err)
	}

	// Ensure 'environments' key exists at the top level
	if _, ok := values["environments"].(map[string]interface{}); !ok {
		return fmt.Errorf("top-level 'environments' key not found or is invalid in values.yaml")
	}

	// Separate common top-level configuration from environment-specific configurations.
	// This map will hold all top-level keys EXCEPT the "environments" block itself.
	commonConfig := make(map[string]interface{})
	for key, val := range values {
		if key != "environments" {
			commonConfig[key] = val
		}
	}

	// Read and parse run.yaml.tmpl template
	tmpl, err := readTemplateFile(templatePath)
	if err != nil {
		return fmt.Errorf("error reading template: %v", err)
	}

	// Create deploy directory
	if err := createDeployDirectory(); err != nil {
		return fmt.Errorf("error creating deploy directory: %v", err)
	}

	// environments block for iteration
	environments, err := getEnvironments(values)
	if err != nil {
		return err
	}

	for envName, envConfigGeneric := range environments {
		envConfig, ok := envConfigGeneric.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid configuration for environment %s: expected a map", envName)
		}

		// Create a final merged configuration for the current environment.
		// Start with common (top-level) values, then override with environment-specific values.
		finalEnvConfig := make(map[string]interface{})
		for k, v := range commonConfig {
			finalEnvConfig[k] = v
		}
		for k, v := range envConfig {
			finalEnvConfig[k] = v
		}

		// --- Validation of critical keys after merge ---
		// These keys are essential for the template and must be present and non-empty
		// in the final merged configuration for each environment.
		mandatoryEnvKeys := []string{"IMAGE_REGISTRY", "SERVICE_NAME"}
		for _, key := range mandatoryEnvKeys {
			val, ok := finalEnvConfig[key].(string)
			if !ok || val == "" {
				return fmt.Errorf("mandatory key '%s' not found or is empty for environment '%s' (after merging values)", key, envName)
			}
		}

		outputFile := filepath.Join(deployDir, fmt.Sprintf("%s.yaml", envName))
		if err := generateYAMLFile(tmpl, outputFile, finalEnvConfig); err != nil {
			return fmt.Errorf("error generating YAML for %s: %v", envName, err)
		}
		fmt.Printf("Generated: %s\n", outputFile)
	}

	return nil
}

// readValuesFile reads and unmarshals the values.yaml file
func readValuesFile(valuesPath string) (map[string]interface{}, error) {
	valuesFile, err := os.ReadFile(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("error reading values.yaml at %s: %v", valuesPath, err)
	}

	var values map[string]interface{}
	err = yaml.Unmarshal(valuesFile, &values)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling values.yaml: %v", err)
	}

	return values, nil
}

// readTemplateFile reads and parses the run.yaml.tmpl template
func readTemplateFile(templatePath string) (*template.Template, error) {
	templateFile, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("error reading template at %s: %v", templatePath, err)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(templateFile))
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %v", err)
	}

	return tmpl, nil
}

// createDeployDirectory creates the deploy directory if it doesn't exist
func createDeployDirectory() error {
	if err := os.MkdirAll(deployDir, 0755); err != nil {
		return fmt.Errorf("error creating deploy directory: %v", err)
	}
	return nil
}

// getEnvironments extracts the environments from the values map
func getEnvironments(values map[string]interface{}) (map[string]interface{}, error) {
	environments, ok := values["environments"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("environments key not found or invalid in values.yaml")
	}
	return environments, nil
}

// generateYAMLFile generates a YAML file for a specific environment
func generateYAMLFile(tmpl *template.Template, outputFile string, envConfig interface{}) error {
	// Create the output file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", outputFile, err)
	}
	defer file.Close()

	// Execute the template with the environment config
	err = tmpl.Execute(file, envConfig)
	if err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}

	return nil
}
