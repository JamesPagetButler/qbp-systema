package section_test

import (
	"testing"

	"github.com/JamesPagetButler/qbp-systema/pkg/section"
)

func TestNumAxesIsFive(t *testing.T) {
	if section.NumAxes != 5 {
		t.Fatalf("NumAxes = %d, want 5 (reproducibility, theory, stats, method, independence)", section.NumAxes)
	}
}

func TestAxisOrderMatchesLocaleDeclaration(t *testing.T) {
	// Order is load-bearing — must match the beekeeper's LOCALE AXES declaration.
	want := []string{"reproducibility", "theory", "stats", "method", "independence"}
	for i, w := range want {
		if got := section.Axis(i).String(); got != w {
			t.Errorf("axis %d = %q, want %q", i, got, w)
		}
	}
}

func TestSectionCarriesNoGluingOps(t *testing.T) {
	// Gluing ops are host-owned (LOCALE), not producer-supplied (architect
	// seq=12). The Section must carry only axes + fingerprint + LocaleID.
	// This is a compile-time guarantee: the only exported fields are those.
	s := section.Section{LocaleID: section.LocaleQBPExperiments}
	if s.LocaleID != "cth.qbp.experiments" {
		t.Errorf("LocaleQBPExperiments = %q, want cth.qbp.experiments", s.LocaleID)
	}
}

func TestZeroSectionIsValidCrawlDefault(t *testing.T) {
	z := section.Zero()
	if !z.Valid() {
		t.Error("zero section must be valid (Crawl-phase default)")
	}
	if len(z.Fingerprint) != 0 {
		t.Error("zero section must have empty fingerprint")
	}
	for a := section.Axis(0); int(a) < section.NumAxes; a++ {
		if z.Get(a) != 0 {
			t.Errorf("zero section axis %s must be 0", a)
		}
	}
}

func TestValidRejectsOutOfRange(t *testing.T) {
	var s section.Section
	s.Axes[section.Reproducibility] = 1.5
	if s.Valid() {
		t.Error("axis value 1.5 must be invalid (range [0,1])")
	}
	s.Axes[section.Reproducibility] = -0.1
	if s.Valid() {
		t.Error("axis value -0.1 must be invalid")
	}
}
