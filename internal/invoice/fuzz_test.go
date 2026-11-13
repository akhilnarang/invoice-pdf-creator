package invoice

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// FuzzConfigPipeline feeds arbitrary YAML through load, validation and HTML
// rendering. Accepted configs must render, and every accepted number must be
// a plain file name.
func FuzzConfigPipeline(f *testing.F) {
	if seed, err := os.ReadFile("../../examples/invoices.example.yaml"); err == nil {
		f.Add(seed)
	}
	f.Add([]byte("invoices:\n  - number: INV-1\n    items:\n      - detail: a\n        rateValue: 19.99\n        qtyValue: 0.333\n"))
	f.Add([]byte("invoices:\n  - number: INV-1\n    total: 500\n    items:\n      - detail: a\n        amount: \"500.00\"\n"))
	f.Add([]byte("invoices:\n  - number: ../x\n---\ninvoices: []\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "invoices.yaml")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadConfig(path)
		if err != nil {
			return
		}
		invoices, err := cfg.ResolveAll()
		if err != nil {
			return
		}
		for _, inv := range invoices {
			if inv.Number == "" || strings.ContainsAny(inv.Number, "./\\: ") || filepath.Base(inv.Number) != inv.Number {
				t.Fatalf("unsafe invoice number accepted: %q", inv.Number)
			}
			if inv.Check() != nil {
				continue
			}
			if _, err := inv.RenderHTML(); err != nil {
				t.Fatalf("valid invoice %q failed to render: %v", inv.Number, err)
			}
		}
	})
}

// FuzzMoney checks that, within the accepted range, Money shows exactly the
// value round2 produces, with grouping commas and two decimals.
func FuzzMoney(f *testing.F) {
	for _, v := range []float64{0, -0.0, 2.675, 19.99 * 0.333, 246810, -5.5, 999999999999999} {
		f.Add("INR", v)
		f.Add("USD", v)
	}
	f.Fuzz(func(t *testing.T, cur string, v float64) {
		s := Money(cur, v)
		if !validAmount(v) {
			return
		}
		dot := strings.IndexByte(s, '.')
		if dot < 0 || len(s)-dot != 3 {
			t.Fatalf("Money(%q, %v) = %q, want two decimals", cur, v, s)
		}
		got, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
		if err != nil {
			t.Fatalf("Money(%q, %v) = %q does not parse: %v", cur, v, s, err)
		}
		if want := round2(v); got != want && !(got == 0 && want == 0) {
			t.Fatalf("Money(%q, %v) = %q, round2 = %v", cur, v, s, want)
		}
	})
}
