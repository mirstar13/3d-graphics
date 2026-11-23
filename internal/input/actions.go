// Package input resolves press events into semantic editor/gameplay actions.
// It does not infer held keys, releases, or timing from terminal repeat events.
package input

import (
	"fmt"
	"strings"
)

// Semantic action IDs are stable keys in project manifests.
const (
	Play     = "action.play"
	Pause    = "action.pause"
	Step     = "action.step"
	Impulse  = "action.impulse"
	Pick     = "action.pick"
	Clear    = "action.clear"
	Next     = "action.next"
	Previous = "action.previous"
	AimUp    = "action.aim_up"
	AimDown  = "action.aim_down"
	AimLeft  = "action.aim_left"
	AimRight = "action.aim_right"
)

// Definition describes a rebindable press action and its focus requirement.
type Definition struct {
	ID, Label    string
	Keys         []string
	ViewportOnly bool
}

var definitions = []Definition{
	{Play, "Play", []string{"p", "f5"}, false}, {Pause, "Pause", []string{"space", "f6"}, false},
	{Step, "Step", []string{"n", "f7"}, false}, {Impulse, "Impulse", []string{"i"}, false},
	{Pick, "Pick", []string{"f"}, false}, {Clear, "Clear", []string{"0"}, false},
	{Next, "Next", []string{"]"}, false}, {Previous, "Previous", []string{"["}, false},
	{AimUp, "Reticle up", []string{"up"}, true}, {AimDown, "Reticle down", []string{"down"}, true},
	{AimLeft, "Reticle left", []string{"left"}, true}, {AimRight, "Reticle right", []string{"right"}, true},
}

// Bindings contains only project overrides; absent actions retain defaults.
type Bindings map[string][]string

// Definitions returns a copy so callers cannot modify shared defaults.
func Definitions() []Definition {
	out := append([]Definition(nil), definitions...)
	for i := range out {
		out[i].Keys = append([]string(nil), out[i].Keys...)
	}
	return out
}

// Keys returns independent effective bindings, falling back to defaults.
func Keys(bindings Bindings, id string) []string {
	if keys, ok := bindings[id]; ok {
		return append([]string(nil), keys...)
	}
	for _, d := range definitions {
		if d.ID == id {
			return append([]string(nil), d.Keys...)
		}
	}
	return nil
}

// Clone copies bindings without sharing editable key slices.
func Clone(b Bindings) Bindings {
	if b == nil {
		return nil
	}
	out := Bindings{}
	for k, v := range b {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// Validate rejects conflicts, unknown actions, and fixed editor/modal keys.
func Validate(b Bindings) error {
	known := map[string]bool{}
	for _, d := range definitions {
		known[d.ID] = true
	}
	for id := range b {
		if !known[id] {
			return fmt.Errorf("unknown input action %s", id)
		}
	}
	reserved := ",a,r,h,b,g,e,k,q,1,2,3,+,=,-,enter,tab,esc,delete,backspace,ctrl+c,ctrl+s,ctrl+z,ctrl+y,ctrl+n,ctrl+o,ctrl+d,ctrl+r,ctrl+u,ctrl+t,f8,"
	used := map[string]string{}
	for _, d := range definitions {
		keys := Keys(b, d.ID)
		if len(keys) == 0 || len(keys) > 3 {
			return fmt.Errorf("%s requires 1..3 keys", d.Label)
		}
		for _, key := range keys {
			arrow := key == "up" || key == "down" || key == "left" || key == "right"
			allowed := len(key) == 1 && strings.Contains("abcdefghijklmnopqrstuvwxyz0123456789[]", key) || key == "space" || key == "f5" || key == "f6" || key == "f7" || key == "f9" || key == "f10" || key == "f11" || key == "f12" || arrow && d.ViewportOnly
			if !allowed || strings.Contains(reserved, ","+key+",") {
				return fmt.Errorf("key %q is reserved or unsupported", key)
			}
			if prior, ok := used[key]; ok {
				return fmt.Errorf("key %q already assigned to %s", key, prior)
			}
			used[key] = d.Label
		}
	}
	return nil
}

// Resolve maps one key event once; viewport actions cannot fire in other panels.
func Resolve(b Bindings, key string, viewport bool) string {
	if key == " " {
		key = "space"
	}
	for _, d := range definitions {
		if d.ViewportOnly && !viewport {
			continue
		}
		for _, binding := range Keys(b, d.ID) {
			if binding == key {
				return d.ID
			}
		}
	}
	return ""
}

// Help formats the current keys for the requested actions in display order.
func Help(b Bindings, ids ...string) string {
	var parts []string
	for _, id := range ids {
		for _, d := range definitions {
			if d.ID == id {
				parts = append(parts, strings.Join(Keys(b, id), "/")+" "+d.Label)
			}
		}
	}
	return strings.Join(parts, " | ")
}
