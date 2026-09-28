package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type MeterType string

const (
	MeterElectricity MeterType = "electricity"
	MeterHeat        MeterType = "heat"
	MeterGas         MeterType = "gas"
	MeterWater       MeterType = "water"
)

type Protocol string

const (
	ProtocolModbus Protocol = "modbus"
	ProtocolMQTT   Protocol = "mqtt"
)

type Meter struct {
	ID       string    `yaml:"id"`
	Type     MeterType `yaml:"type"`
	Protocol Protocol  `yaml:"protocol"`
	Address  string    `yaml:"address,omitempty"`
	Topic    string    `yaml:"topic,omitempty"`
	Carrier  string    `yaml:"carrier,omitempty"`
}

type Building struct {
	ID      string  `yaml:"id"`
	Name    string  `yaml:"name"`
	Meters  []Meter `yaml:"meters"`
	HasHeat bool    `yaml:"has_heat,omitempty"`
}

type Root struct {
	Site      string     `yaml:"site"`
	Buildings []Building `yaml:"buildings"`
}

func Load(path string) (*Root, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var root Root
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	return &root, nil
}

type ValidationIssue struct {
	Building string
	Meter    string
	Message  string
}

func (r *Root) Validate() []ValidationIssue {
	var issues []ValidationIssue
	if strings.TrimSpace(r.Site) == "" {
		issues = append(issues, ValidationIssue{Message: "site name is required"})
	}
	if len(r.Buildings) == 0 {
		issues = append(issues, ValidationIssue{Message: "at least one building is required"})
	}
	seenBuilding := map[string]struct{}{}
	for _, b := range r.Buildings {
		if strings.TrimSpace(b.ID) == "" {
			issues = append(issues, ValidationIssue{Building: b.Name, Message: "building id is required"})
			continue
		}
		if _, dup := seenBuilding[b.ID]; dup {
			issues = append(issues, ValidationIssue{Building: b.ID, Message: "duplicate building id"})
		}
		seenBuilding[b.ID] = struct{}{}
		if len(b.Meters) == 0 {
			issues = append(issues, ValidationIssue{Building: b.ID, Message: "building has no meters"})
		}
		typesPresent := map[MeterType]bool{}
		seenMeter := map[string]struct{}{}
		for _, m := range b.Meters {
			mid := m.ID
			if mid == "" {
				mid = "(unnamed)"
			}
			if _, dup := seenMeter[m.ID]; m.ID != "" && dup {
				issues = append(issues, ValidationIssue{Building: b.ID, Meter: m.ID, Message: "duplicate meter id"})
			}
			if m.ID != "" {
				seenMeter[m.ID] = struct{}{}
			}
			if err := validateMeter(m); err != nil {
				issues = append(issues, ValidationIssue{Building: b.ID, Meter: mid, Message: err.Error()})
			}
			typesPresent[m.Type] = true
		}
		if !typesPresent[MeterElectricity] {
			issues = append(issues, ValidationIssue{
				Building: b.ID,
				Message:  "EnEfG readiness: electricity meter required per building",
			})
		}
		needsHeat := b.HasHeat || typesPresent[MeterHeat]
		if needsHeat && !typesPresent[MeterHeat] {
			issues = append(issues, ValidationIssue{
				Building: b.ID,
				Message:  "EnEfG readiness: heat meter required when building has heating",
			})
		}
	}
	return issues
}

func validateMeter(m Meter) error {
	switch m.Type {
	case MeterElectricity, MeterHeat, MeterGas, MeterWater:
	default:
		return fmt.Errorf("invalid meter type %q (want electricity|heat|gas|water)", m.Type)
	}
	switch m.Protocol {
	case ProtocolModbus:
		if strings.TrimSpace(m.Address) == "" {
			return fmt.Errorf("modbus meter requires address")
		}
	case ProtocolMQTT:
		if strings.TrimSpace(m.Topic) == "" {
			return fmt.Errorf("mqtt meter requires topic")
		}
	default:
		return fmt.Errorf("invalid protocol %q (want modbus|mqtt)", m.Protocol)
	}
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("meter id is required")
	}
	return nil
}
