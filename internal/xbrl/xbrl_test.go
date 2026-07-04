package xbrl

import "testing"

// synthetic instance modeled on the real IDX/Fujitsu output: two plain facts
// (Assets, ProfitLoss), one dimensional Assets fact that must be filtered out,
// and dei metadata.
const sample = `<?xml version="1.0"?>
<xbrl xmlns:link="http://www.xbrl.org/2003/linkbase"
      xmlns="http://www.xbrl.org/2003/instance"
      xmlns:idx-cor="http://www.idx.co.id/xbrl/taxonomy/2020-01-01/cor"
      xmlns:idx-dei="http://www.idx.co.id/xbrl/taxonomy/2020-01-01/dei"
      xmlns:xbrldi="http://xbrl.org/2006/xbrldi">
  <context id="CurrentYearInstant">
    <entity><identifier>x_user</identifier></entity>
    <period><instant>2026-03-31</instant></period>
  </context>
  <context id="CurrentYearDuration">
    <entity><identifier>x_user</identifier></entity>
    <period><startDate>2026-01-01</startDate><endDate>2026-03-31</endDate></period>
  </context>
  <context id="CurrentYearInstant_Segment">
    <entity><identifier>x_user</identifier></entity>
    <period><instant>2026-03-31</instant></period>
    <scenario><xbrldi:explicitMember dimension="idx-cor:SegmentAxis">idx-cor:SomeSegment</xbrldi:explicitMember></scenario>
  </context>
  <idx-dei:EntityName contextRef="CurrentYearInstant">PT Test Tbk.</idx-dei:EntityName>
  <idx-dei:EntityCode contextRef="CurrentYearInstant">TEST</idx-dei:EntityCode>
  <idx-cor:Assets contextRef="CurrentYearInstant" unitRef="IDR">1000000</idx-cor:Assets>
  <idx-cor:Assets contextRef="CurrentYearInstant_Segment" unitRef="IDR">400000</idx-cor:Assets>
  <idx-cor:ProfitLoss contextRef="CurrentYearDuration" unitRef="IDR">250000</idx-cor:ProfitLoss>
</xbrl>`

func TestParse(t *testing.T) {
	rep, err := ParseBytes([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if rep.Entity.Name != "PT Test Tbk." || rep.Entity.Code != "TEST" {
		t.Errorf("entity = %+v, want name/code TEST", rep.Entity)
	}

	// Dimensional Assets (400000) must be excluded; only two plain facts remain.
	if len(rep.Accounts) != 2 {
		t.Fatalf("got %d accounts, want 2 (dimensional fact should be filtered): %+v", len(rep.Accounts), rep.Accounts)
	}

	assets := findAccount(rep, "Assets")
	if assets == nil {
		t.Fatal("Assets not found")
	}
	if assets.NumericIDR == nil || *assets.NumericIDR != 1000000 {
		t.Errorf("Assets numeric = %v, want 1000000", assets.NumericIDR)
	}
	if assets.Period != "as of 2026-03-31" {
		t.Errorf("Assets period = %q", assets.Period)
	}

	pl := findAccount(rep, "ProfitLoss")
	if pl == nil || pl.NumericIDR == nil || *pl.NumericIDR != 250000 {
		t.Errorf("ProfitLoss = %+v, want 250000", pl)
	}
	if pl.Period != "2026-01-01 .. 2026-03-31" {
		t.Errorf("ProfitLoss period = %q", pl.Period)
	}
}

func findAccount(r *Report, concept string) *Account {
	for i := range r.Accounts {
		if r.Accounts[i].Concept == concept {
			return &r.Accounts[i]
		}
	}
	return nil
}
