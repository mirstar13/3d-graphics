package input

import (
	"strings"
	"testing"
)

func TestBindingsDefaultsAndValidation(t *testing.T) {
	if err := Validate(nil); err != nil {
		t.Fatal(err)
	}
	if Resolve(nil, "p", false) != Play || Resolve(nil, " ", false) != Pause || Resolve(nil, "up", false) != "" || Resolve(nil, "up", true) != AimUp {
		t.Fatal("default focus resolution")
	}
	b := Bindings{Play: {"v"}, AimUp: {"w"}}
	if err := Validate(b); err != nil {
		t.Fatal(err)
	}
	if Resolve(b, "p", false) != "" || Resolve(b, "v", false) != Play || Resolve(b, "w", false) != "" || Resolve(b, "w", true) != AimUp {
		t.Fatal("remap did not replace defaults")
	}
	if !strings.Contains(Help(b, Play), "v Play") {
		t.Fatal("help stale")
	}
	for _, bad := range []Bindings{{Play: {"q"}}, {Play: {"up"}}, {Play: {"i"}}, {Play: {}}, {Play: {"v", "v"}}, {"unknown": {"v"}}, {Play: {"ctrl+c"}}, {Play: {"P"}}} {
		if Validate(bad) == nil {
			t.Fatal("invalid bindings accepted", bad)
		}
	}
	clone := Clone(b)
	clone[Play][0] = "x"
	keys := Keys(b, Play)
	keys[0] = "x"
	if b[Play][0] != "v" {
		t.Fatal("binding alias")
	}
	defs := Definitions()
	defs[0].Keys[0] = "x"
	if Resolve(nil, "p", false) != Play {
		t.Fatal("default alias")
	}
}
