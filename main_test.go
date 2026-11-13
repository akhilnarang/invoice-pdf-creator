package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"invoice-pdf-creator/internal/invoice"
)

var stringsSink []string

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" INV-01, ,INV-02 ")
	if len(got) != 2 || got[0] != "INV-01" || got[1] != "INV-02" {
		t.Fatalf("splitCSV returned %#v", got)
	}
}

func TestGenConfigDefaultsToInvoicesYAML(t *testing.T) {
	if got := newOptions().config; got != "invoices.yaml" {
		t.Fatalf("default config = %q, want invoices.yaml", got)
	}
	if err := runGen([]string{"--config", ""}); err == nil {
		t.Fatal("runGen accepted an empty --config")
	}
}

func TestGenNumberSelectsOneInvoice(t *testing.T) {
	out := t.TempDir()
	err := runGen([]string{"--config", "examples/invoices.example.yaml", "--num", "2", "--out", out, "--html-only"})
	if err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(out, "*"))
	if len(files) != 1 || filepath.Base(files[0]) != "INV-0000-02.html" {
		t.Fatalf("wrote %v, want only INV-0000-02.html", files)
	}
}

func TestGenUnknownNumberFails(t *testing.T) {
	out := t.TempDir()
	err := runGen([]string{"--config", "examples/invoices.example.yaml", "--only", "INV-0000-01,INV-9999-99", "--out", out, "--html-only"})
	if err == nil {
		t.Fatal("runGen accepted an invoice number missing from the config")
	}
	if entries, _ := os.ReadDir(out); len(entries) != 0 {
		t.Fatalf("wrote %d files before failing", len(entries))
	}
}

func TestSelectInvoices(t *testing.T) {
	invs := []invoice.Invoice{{Number: "INV-0001-19"}, {Number: "INV-0002-03"}, {Number: "INV-0002-19"}, {Number: "INV-0002-20"}}
	numbers := func(got []invoice.Invoice) string {
		var s []string
		for _, inv := range got {
			s = append(s, inv.Number)
		}
		return strings.Join(s, ",")
	}
	ok := []struct {
		want []string
		out  string
	}{
		{[]string{"INV-0002-19"}, "INV-0002-19"},
		{[]string{"20", "3"}, "INV-0002-03,INV-0002-20"},
		{[]string{"03", "INV-0002-03"}, "INV-0002-03"},
	}
	for _, tc := range ok {
		got, err := selectInvoices(invs, tc.want)
		if err != nil || numbers(got) != tc.out {
			t.Errorf("selectInvoices(%v) = %q, %v; want %q", tc.want, numbers(got), err, tc.out)
		}
	}
	zero, err := selectInvoices([]invoice.Invoice{{Number: "INV-0002-00"}}, []string{"0"})
	if err != nil || len(zero) != 1 {
		t.Errorf("selectInvoices(0) = %v, %v; want INV-0002-00", zero, err)
	}
	for _, bad := range []string{"19", "21", "2026"} {
		if _, err := selectInvoices(invs, []string{bad}); err == nil {
			t.Errorf("selectInvoices(%q) did not fail", bad)
		}
	}
}

func TestParseOnlyTakesSeveralArgs(t *testing.T) {
	o := newOptions()
	if err := o.parse([]string{"--only", "19", "--num", "22", "--out", "x", "20,21", "23"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(o.only, ","); got != "19,22,20,21,23" || o.out != "x" {
		t.Fatalf("only = %q, out = %q", got, o.out)
	}
}

func TestParseRejectsBadValues(t *testing.T) {
	for _, args := range [][]string{
		{"--only", ", ,"},
		{"--num", ""},
		{"--only", "--out", "x"},
		{"--total", "0"},
		{"--total", "NaN"},
		{"--total", "Inf"},
		{"19", "--out", "x"},
		{"19", ","},
		{"--bogus"},
		{"--config"},
	} {
		o := newOptions()
		if err := o.parse(args); err == nil {
			t.Errorf("parse(%q) accepted", args)
		}
	}
}

func BenchmarkSplitCSV(b *testing.B) {
	for b.Loop() {
		stringsSink = splitCSV("INV-0000-01,INV-0000-02,INV-0000-03")
	}
}
