package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Paths struct {
	Stacks string
	Vtl    string
	Schema string
}

type Config struct {
	Stacks []Stack
	Vtl    map[string]*string
	Schema string
}

type Stack struct {
	StackName string
	Vars      StackVars
}

type StackVars struct {
	Name      string        `json:"name"`
	IamPath   string        `json:"iamPath"`
	Regions   []string      `json:"regions"`
	Backend   VarsBackend   `json:"backend"`
	Artifacts VarsArtifacts `json:"artifacts"`
	Domain    VarsDomain    `json:"domain"`
	Alarms    VarsAlarms    `json:"alarms"`
	Groups    VarsGroups    `json:"groups"`
}

type VarsBackend struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
	Region string `json:"region"`
	Table  string `json:"table"`
}

type VarsArtifacts struct {
	BucketPrefix string `json:"bucketPrefix"`
	ObjectPrefix string `json:"objectPrefix"`
}

type VarsDomain struct {
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
}

type VarsAlarms struct {
	Enabled bool     `json:"enabled"`
	SendTo  []string `json:"sendTo"`
}

type VarsGroups struct {
	InfraAdmin string `json:"infraAdmin"`
}

func (paths Paths) LoadConfig() (cfg Config, err error) {
	if cfg.Stacks, err = paths.loadStacks(); err != nil {
		return cfg, fmt.Errorf("Failed to load stacks: %w", err)
	} else if cfg.Vtl, err = paths.loadVtl(); err != nil {
		return cfg, fmt.Errorf("Failed to load vtl: %w", err)
	} else if cfg.Schema, err = paths.loadSchema(); err != nil {
		return cfg, fmt.Errorf("Failed to load schema: %w", err)
	}
	return cfg, nil
}

func (paths Paths) loadStacks() ([]Stack, error) {
	stacks := []Stack{}
	processFile := func(filename string, contents []byte) error {
		stack := Stack{StackName: strings.TrimSuffix(filename, ".json")}
		if err := json.Unmarshal(contents, &stack.Vars); err != nil {
			return fmt.Errorf("Invalid json: %w", err)
		}

		// todo: validate vars
		stacks = append(stacks, stack)
		return nil
	}

	return stacks, processDir(paths.Stacks, ".json", processFile)
}

func (paths Paths) loadVtl() (map[string]*string, error) {
	templates := map[string]*string{}
	processFile := func(filename string, contents []byte) error {
		s := string(contents)
		templates[filename] = &s
		return nil
	}

	return templates, processDir(paths.Vtl, ".vm", processFile)
}

func (paths Paths) loadSchema() (string, error) {
	schema := ""
	processFile := func(filename string, contents []byte) error {
		schema = fmt.Sprintf("%s\n\n#%s\n\n%s", schema, filename, string(contents))
		return nil
	}

	return schema, processDir(paths.Schema, ".graphql", processFile)
}

func processDir(dir string, suffix string, processFile func(filename string, contents []byte) error) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("Failed to read dir: %w", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), suffix) {
			filename := filepath.Join(dir, file.Name())
			contents, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("Failed to read file: %w", err)
			} else if err = processFile(filename, contents); err != nil {
				return fmt.Errorf("Failed to process file '%s': %w", filename, err)
			}
		}
	}
	return nil
}

// this is important as lock tables need to be processed in order
func (cfg StackVars) OrderedRegions() []string {
	return append([]string{"us-east-1"}, cfg.Regions...)
}

func (domain VarsDomain) RegionalUrlTemplate() string {
	return fmt.Sprintf("https://<region>.%s/graphl", domain.Fqdn())
}

func (domain VarsDomain) Fqdn() string {
	if domain.Subdomain != "" {
		return fmt.Sprintf("%s.%s", domain.Subdomain, domain.Name)
	}
	return domain.Name
}
