package input

import "testing"

func TestPointerOriginAfterCaptureAndResize(t *testing.T) {
	var p Pointer
	p.Motion(30, 40)
	p.Reset(true)
	p.Motion(70, 60)
	p.Motion(77, 58)
	p.Motion(74, 57)
	if dx, dy := p.Take(); dx != 4 || dy != -3 {
		t.Fatal(dx, dy)
	}
	if dx, dy := p.Take(); dx != 0 || dy != 0 {
		t.Fatal(dx, dy)
	}
	p.Motion(2, 3)
	p.Reset(true)
	p.Motion(10, 20)
	if dx, dy := p.Take(); dx != 0 || dy != 0 {
		t.Fatal("resize jump", dx, dy)
	}
}
