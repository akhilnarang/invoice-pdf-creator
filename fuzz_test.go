package main

import (
	"strings"
	"testing"

	"invoice-pdf-creator/internal/invoice"
)

// FuzzParse checks the flag parser never panics and never keeps an empty
// selection.
func FuzzParse(f *testing.F) {
	f.Add("--only\x0019\x00--num\x0022\x00--out\x00x\x0020,21")
	f.Add("--only\x00, ,")
	f.Add("--total\x00NaN")
	f.Add("--only")
	f.Fuzz(func(t *testing.T, joined string) {
		o := newOptions()
		if err := o.parse(strings.Split(joined, "\x00")); err != nil {
			return
		}
		for _, n := range o.only {
			if strings.TrimSpace(n) == "" || strings.Contains(n, ",") {
				t.Fatalf("bad selector %q in %q", n, o.only)
			}
		}
	})
}

// FuzzSelectInvoices checks selection returns a duplicate-free, in-order
// subset where every want maps to a matching invoice.
func FuzzSelectInvoices(f *testing.F) {
	f.Add("INV-0001-19,INV-0002-03,INV-0002-19,INV-0002-00", "3,0")
	f.Add("INV-1,INV-01", "1")
	f.Fuzz(func(t *testing.T, numbers, wants string) {
		var invs []invoice.Invoice
		for _, n := range splitCSV(numbers) {
			invs = append(invs, invoice.Invoice{Number: n})
		}
		ws := splitCSV(wants)
		got, err := selectInvoices(invs, ws)
		if err != nil {
			return
		}
		pos := -1
		for _, g := range got {
			i := -1
			for j := pos + 1; j < len(invs); j++ {
				if invs[j].Number == g.Number {
					i = j
					break
				}
			}
			if i < 0 {
				t.Fatalf("result %v not an in-order subset of %v", got, invs)
			}
			pos = i
		}
		for _, w := range ws {
			ok := false
			for _, g := range got {
				ok = ok || g.Number == w || sameSuffix(g.Number, w)
			}
			if !ok {
				t.Fatalf("want %q not satisfied by %v", w, got)
			}
		}
	})
}
