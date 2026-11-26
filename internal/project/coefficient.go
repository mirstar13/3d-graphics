package project

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Coefficient is an optional collider material coefficient. Its zero value
// inherits the physics world's body descriptor default. It is copied by value
// with entity snapshots, including explicit zero-friction or zero-bounce edits.
type Coefficient struct {
	value float64
	set   bool
}

// Resolve returns the authored coefficient or the supplied physics default.
func (c Coefficient) Resolve(fallback float64) float64 {
	if c.set {
		return c.value
	}
	return fallback
}

// IsZero permits YAML to omit inherited values.
func (c Coefficient) IsZero() bool { return !c.set }

// MarshalYAML stores an authored coefficient as a numeric scalar.
func (c Coefficient) MarshalYAML() (any, error) {
	if !c.set {
		return nil, nil
	}
	return c.value, nil
}

// UnmarshalYAML restores an explicit numeric coefficient, including zero.
func (c *Coefficient) UnmarshalYAML(node *yaml.Node) error {
	var value float64
	if err := node.Decode(&value); err != nil {
		return err
	}
	*c = Coefficient{value, true}
	return nil
}

func coefficientProperty(key, label string, field func(*Entity) *Coefficient) Property {
	return Property{Key: key, Label: label, Get: func(e Entity) string {
		c := field(&e)
		if !c.set {
			return "default"
		}
		return strconv.FormatFloat(c.value, 'g', -1, 64)
	}, Set: func(e *Entity, text string) error {
		if text == "default" {
			*field(e) = Coefficient{}
			return nil
		}
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return fmt.Errorf("%s requires a number or default", label)
		}
		*field(e) = Coefficient{value, true}
		return nil
	}}
}
