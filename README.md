# runparcel

runparcel is a CLI tool for managing Cloud Run deployments across multiple environments.

## Features

- Generate deployment YAML files for different environments.
- Manage configurations using a single `run.yaml` and `values.yaml` file.

## Installation

```bash
go install github.com/runparcel/runparcel/cmd/runparcel
```

## Usage
### Prepare Your Files

`values.yaml`: Define environment-specific configurations.
```yaml
# Top-level image registry configuration
IMAGE_REGISTRY: us-central1-docker.pkg.dev/my-project/my-gar
SERVICE_NAME: myapp

environments:
  dev:
    SERVICE_NAME: "myapp-dev"
    REGION: "us-central1"
    IMAGE: "gcr.io/myapp/dev:latest"
    MIN_INSTANCES: 1
    MAX_INSTANCES: 3
    CPU_LIMIT: "1"
    MEMORY_LIMIT: "512Mi"
    PERCENT_TO_LATEST: 100
    ENV_VARS:
      - name: "ENV"
        value: "dev"
  prd:
    SERVICE_NAME: "myapp-prd"
    REGION: "us-central1"
    IMAGE: "gcr.io/myapp/prd:latest"
    MIN_INSTANCES: 3
    MAX_INSTANCES: 10
    CPU_LIMIT: "2"
    MEMORY_LIMIT: "1Gi"
    PERCENT_TO_LATEST: 100
    ENV_VARS:
      - name: "ENV"
        value: "prd"
```

`run.yaml`: Define the CloudRun YAML template.
```
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: {{ .SERVICE_NAME }}
  labels:
    cloud.googleapis.com/location: {{ .REGION }}
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: {{ .MIN_INSTANCES }}
        autoscaling.knative.dev/maxScale: {{ .MAX_INSTANCES }}
    spec:
      containers:
      - image: {{ .IMAGE_REGISTRY }}/{{ .SERVICE_NAME }}:{{ .TAG }}
        env:
        {{- range .ENV_VARS }}
        - name: {{ .name }}
          value: {{ .value }}
        {{- end }}
        resources:
          limits:
            cpu: {{ .CPU_LIMIT }}
            memory: {{ .MEMORY_LIMIT }}
  traffic:
  - percent: {{ .PERCENT_TO_LATEST }}
    latestRevision: true
```

### Generate Deployment YAMLs
Run the generate command to create environment-specific YAML files.

#### Default Paths
If `run.yaml` is located in `cloudrun/run.yaml` and `values.yaml` is in the root directory:
```bash
runparcel generate
```

#### Custom Paths
Specify custom paths for `run.yaml` and `values.yaml`:
```bash
runparcel generate --template /path/to/run.yaml --values /path/to/values.yaml
```

#### Output
The generated YAML files will be saved in the `deploy/` directory:

```bash
deploy/
├── dev.yaml
└── prd.yaml
```

#### Example Generated YAML
For the dev environment, `deploy/dev.yaml` will look like this:
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: myapp-dev
  labels:
    cloud.googleapis.com/location: us-central1
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: 1
        autoscaling.knative.dev/maxScale: 3
    spec:
      containers:
      - image: gcr.io/myapp/dev:latest
        env:
        - name: ENV
          value: dev
        resources:
          limits:
            cpu: "1"
            memory: "512Mi"
  traffic:
  - percent: 100
    latestRevision: true
```

### Commands
| Command   | Description                                      |
|-----------|--------------------------------------------------|
| `generate` | Generate deployment YAML files.                 |
| `lint`     | Validate the generated YAML files (coming soon).|
| `tag`      | Generate a deployment tag (coming soon).        |


### Contributing
Pull requests are welcome! For major changes, please open an issue first to discuss what you’d like to change.