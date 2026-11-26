package project

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// OptionalNumber stores an authored scalar by value so snapshots retain both
// explicit zeroes and the distinction between an override and inheritance.
type OptionalNumber[T float64 | uint32] struct {
	value T
	set   bool
}

// Coefficient is an body damping or collider material coefficient with an inherited default.
type Coefficient = OptionalNumber[float64]

// CollisionBits is a 32-bit layer or mask override.
type CollisionBits = OptionalNumber[uint32]

// Resolve returns the authored number or the supplied physics default.
func (c OptionalNumber[T]) Resolve(fallback T) T {
	if c.set {
		return c.value
	}
	return fallback
}

// IsZero permits YAML to omit inherited values.
func (c OptionalNumber[T]) IsZero() bool { return !c.set }

// MarshalYAML stores an authored number as a numeric scalar.
func (c OptionalNumber[T]) MarshalYAML() (any, error) {
	if !c.set {
		return nil, nil
	}
	return c.value, nil
}

// UnmarshalYAML restores an explicit numeric number, including zero.
func (c *OptionalNumber[T]) UnmarshalYAML(node *yaml.Node) error {
	var value T
	if _, bits := any(value).(uint32); bits && node.ShortTag() != "!!int" {
		return fmt.Errorf("collision bits require an unsigned 32-bit integer")
	}
	if err := node.Decode(&value); err != nil {
		return err
	}
	*c = OptionalNumber[T]{value, true}
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

// bitsProperty displays hexadecimal so individual collision channels are visible.
func bitsProperty(key, label string, field func(*Entity) *CollisionBits) Property {
	return Property{Key: key, Label: label, Get: func(e Entity) string {
		bits := field(&e)
		if !bits.set {
			return "default"
		}
		return fmt.Sprintf("0x%08X", bits.value)
	}, Set: func(e *Entity, text string) error {
		if text == "default" {
			*field(e) = CollisionBits{}
			return nil
		}
		base := 10
		if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
			text = text[2:]
			base = 16
		}
		value, err := strconv.ParseUint(text, base, 32)
		if err != nil {
			return fmt.Errorf("%s requires a 32-bit decimal or hexadecimal integer, or default", label)
		}
		*field(e) = CollisionBits{uint32(value), true}
		return nil
	}}
}
