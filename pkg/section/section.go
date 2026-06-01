// Package section defines the sheaf-trust section a QBP producer attaches to
// each trust assertion it emits into the Confluent-Trust hypergraph.
//
// Per the multi-tenant federation emulation (qbp-architecture, cth-qbp-live-testing
// seq=7/8/10) the CTH trust model is a *sheaf section over a locale*, not a
// scalar/quaternion/trajectory score. Trust has five axes; the CTH host glues
// member sections per-axis (meet for most, join for theory) to compute cluster
// trust. The *producer* (this QBP scout / bookkeeper) supplies the section
// values on each assertion — the host cannot reconstruct provenance it never saw.
//
// Crawl/Sprint 3 stance (architect seq=10 Q3): carry the fields, zero-valued.
// The gluing itself does not activate until the CTH scoring-spec rewrite ships
// (joint qbp-architecture + cth-implementor work). This package is the
// forward-compatible producer-side shape; the host-side gluing lives in CTH.
package section

// Axis indexes the five trust axes. Order is load-bearing: it matches the
// LOCALE AXES declaration in the beekeeper's formulation
// (cth.qbp.experiments AXES [reproducibility, theory, stats, method, independence]).
type Axis int

const (
	Reproducibility Axis = iota // glue: meet — one weak joint poisons the claim
	Theory                      // glue: join — any strong theoretical anchor elevates
	Stats                       // glue: meet
	Method                      // glue: meet
	Independence                // glue: meet — relational; see SourceFingerprint
	numAxes
)

// NumAxes is the section width (5).
const NumAxes = int(numAxes)

// AxisName maps an Axis to its canonical locale-declaration name.
func (a Axis) String() string {
	switch a {
	case Reproducibility:
		return "reproducibility"
	case Theory:
		return "theory"
	case Stats:
		return "stats"
	case Method:
		return "method"
	case Independence:
		return "independence"
	default:
		return "unknown"
	}
}

// Gluing operations (meet/join per axis) deliberately do NOT live here.
// They are type-level properties of the LOCALE, owned and applied by the CTH
// host — not instance-level data a producer supplies (architect cth-qbp-live-testing
// seq=12; LOCALE/CONTRACT separation per live-test seq=374/375). If the host
// read gluing ops off incoming sections, a buggy/inconsistent producer could
// inject wrong ops (theory-join on one section, theory-meet on another) and the
// host would silently glue with whatever it received. So a Section carries only
// its axis values + provenance + the LOCALE it was produced under; the host
// validates the locale matches its own and applies the canonical gluing.

// LocaleQBPExperiments is the QBP experiments locale identifier — the locale
// QBP-domain trust assertions are produced under. The CTH host validates an
// incoming section's LocaleID against its own LOCALE definition and rejects on
// mismatch (fail-early), rather than gluing with mismatched ops.
const LocaleQBPExperiments = "cth.qbp.experiments"

// Section is the producer-side sheaf-trust section attached to one assertion.
//
//	Axes:        per-axis trust values in [0,1]; index by Axis constants.
//	Fingerprint: source provenance fingerprint for the Independence axis.
//	             Sprint 3 = Jaccard proxy over {lab, equipment, funding};
//	             Walk = formal Grothendieck-site cover (independence is a
//	             relational property between sections, so a per-section value
//	             needs either the proxy or the site topology — architect seq=7 Q).
//	LocaleID:    the LOCALE this section was produced under (e.g.
//	             LocaleQBPExperiments). The CTH host validates it matches its
//	             own LOCALE and rejects on mismatch — it does NOT carry gluing
//	             ops (those are host-owned; architect seq=12).
//
// Zero value is a valid "no provenance asserted yet" section (all axes 0,
// empty fingerprint, no locale) — the Crawl-phase default until the scoring
// spec activates.
type Section struct {
	Axes        [NumAxes]float64
	Fingerprint []byte
	LocaleID    string
}

// Zero returns the Crawl-phase default section (all axes zero, no fingerprint,
// no locale).
func Zero() Section { return Section{} }

// Valid reports whether every axis value is in [0,1].
func (s Section) Valid() bool {
	for _, v := range s.Axes {
		if v < 0 || v > 1 {
			return false
		}
	}
	return true
}

// Get returns the value on a single axis.
func (s Section) Get(a Axis) float64 { return s.Axes[a] }
