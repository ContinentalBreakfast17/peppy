package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Stack struct {
	StackName string
	Vars      StackVars
}

type StackVars struct {
	Name      string        `json:"name"`
	IamPath   string        `json:"iamPath"`
	Regions   []string      `json:"region"`
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
	Name string `json:"name"`
}

type VarsAlarms struct {
	Enabled bool     `json:"enabled"`
	SendTo  []string `json:"sendTo"`
}

type VarsGroups struct {
	InfraAdmin string `json:"infraAdmin"`
}

func LoadStacks(path string) ([]Stack, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read dir: %w", err)
	}

	stacks := []Stack{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			filename := filepath.Join(path, file.Name())
			contents, err := os.ReadFile(filename)
			if err != nil {
				return nil, fmt.Errorf("Failed to read file: %w", err)
			}

			stack := Stack{StackName: strings.TrimSuffix(file.Name(), ".json")}
			if err := json.Unmarshal(contents, &stack.Vars); err != nil {
				return nil, fmt.Errorf("Failed to read json file '%s': %w", filename, err)
			}

			// todo: validate vars

			stacks = append(stacks, stack)
		}
	}

	return stacks, nil
}
