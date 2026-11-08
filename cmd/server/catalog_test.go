package main

import (
	"strings"
	"testing"

	"painel-economico/internal/bcb"
	"painel-economico/internal/ibge"
	"painel-economico/internal/indicator"
)

func TestCatalogIsComplete(t *testing.T) {
	defs := catalog(bcb.New("", nil), ibge.New("", nil))
	seen := map[string]bool{}
	for _, d := range defs {
		if seen[d.ID] {
			t.Errorf("duplicate indicator id %q", d.ID)
		}
		seen[d.ID] = true

		if d.Name == "" || d.ShortName == "" || d.Unit == "" || d.Description == "" || d.Source == "" {
			t.Errorf("%s: every descriptive field must be filled: %+v", d.ID, d.Indicator)
		}
		if d.Frequency != indicator.Daily && d.Frequency != indicator.Monthly {
			t.Errorf("%s: unknown frequency %q", d.ID, d.Frequency)
		}
		if !strings.HasPrefix(d.SourceURL, "https://") {
			t.Errorf("%s: source URL %q must use https", d.ID, d.SourceURL)
		}
		if d.Upstream == nil {
			t.Errorf("%s: no upstream source", d.ID)
		}
	}
}
