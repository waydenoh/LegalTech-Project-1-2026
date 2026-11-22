package playbook

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed default_sg_tenancy.json
var defaultTenancyJSON []byte

func DefaultPlaybook() (*Playbook, error) {
	var pb Playbook
	if err := json.Unmarshal(defaultTenancyJSON, &pb); err != nil {
		return nil, fmt.Errorf("parse embedded default playbook: %w", err)
	}
	return &pb, nil
}

func LoadFromFile(path string) (*Playbook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read playbook file: %w", err)
	}
	var pb Playbook
	if err := json.Unmarshal(data, &pb); err != nil {
		return nil, fmt.Errorf("unmarshal playbook json: %w", err)
	}
	return &pb, nil
}
