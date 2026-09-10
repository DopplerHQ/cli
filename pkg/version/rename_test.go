package version

import "testing"

func TestIsRenamed(t *testing.T) {
	orig := ProgramName
	defer func() { ProgramName = orig }()

	ProgramName = "doppler"
	if IsRenamed() {
		t.Error("the official doppler build must not report as renamed")
	}
	ProgramName = "doppler-agent"
	if !IsRenamed() {
		t.Error("a rebranded build (doppler-agent) must report as renamed")
	}
}
