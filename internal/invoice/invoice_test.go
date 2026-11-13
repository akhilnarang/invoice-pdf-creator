package invoice

import (
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var moneySink string
var htmlSink string
var invoicesSink []Invoice

func TestMoney(t *testing.T) {
	cases := []struct {
		cur  string
		in   float64
		want string
	}{
		// INR — Indian digit grouping.
		{"INR", 246810.00, "2,46,810.00"},
		{"INR", 98765.40, "98,765.40"},
		{"INR", 24680.50, "24,680.50"},
		{"INR", 1250.50, "1,250.50"},
		{"INR", 815000.00, "8,15,000.00"},
		{"INR", 123456789.01, "12,34,56,789.01"},
		{"INR", 100.00, "100.00"},
		// USD / others — standard 3-digit grouping.
		{"USD", 3141.50, "3,141.50"},
		{"USD", 2718.30, "2,718.30"},
		{"USD", 1000.00, "1,000.00"},
		{"USD", 1234567.89, "1,234,567.89"},
		{"USD", 123456.78, "123,456.78"},
		{"USD", -5.50, "-5.50"},
		{"INR", 0, "0.00"},
		{"INR", -0.0, "0.00"},
		// Exactly-representable two-decimal values round-trip cleanly.
		{"USD", 0.25, "0.25"},
		{"USD", 10.10, "10.10"},
		// Rounding to 2dp uses the actual stored float (strconv's correct
		// rounding). 2.675 is stored just below 2.675, so it rounds to 2.67;
		// this documents the float-reality behaviour rather than hiding it.
		{"USD", 2.675, "2.67"},
		{"USD", 100.255, "100.25"},
	}
	for _, c := range cases {
		got := Money(c.cur, c.in)
		if got != c.want {
			t.Errorf("Money(%q, %.2f) = %q, want %q", c.cur, c.in, got, c.want)
		}
	}
}

func TestRenderHTMLContainsKeyFields(t *testing.T) {
	inv := Invoice{
		Number:   "INV-0000-03",
		Date:     "05 Jan, 2026",
		Due:      "05 Jan, 2026",
		Currency: "INR",
		Seller: Party{
			Name:    "Jane Contractor",
			Email:   "jane@example.com",
			Address: []string{"123 Example Street"},
			TaxIDs:  []string{"GSTIN: 27ABCDE1234F1Z5", "PAN: ABCDE1234F"},
		},
		BilledTo: Party{Name: "Acme Corp", Address: []string{"1 Market Plaza, Suite 100"}},
		Items: []LineItem{
			{Detail: "Contractor Hourly", Qty: "80", RateValue: 1250.50, QtyValue: 80, RateUnit: "/hrs"},
		},
		Payment: []PayRow{{Label: "Bank Name:", Value: "EXAMPLE BANK"}},
		Note:    "Supply meant for export under bond or Letter of Undertaking without payment of integrated tax vide LUT No. AD0000000000000.",
	}
	html, err := inv.RenderHTML()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"INV-0000-03",
		"jane@example.com",
		"GSTIN: 27ABCDE1234F1Z5",
		"Rate (INR)",
		"1,250.50/hrs",
		"INR&nbsp;1,00,040.00",
		"EXAMPLE BANK",
		"vide LUT No. AD0000000000000",
		"Tax Invoice",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered HTML missing %q", want)
		}
	}
	// Branding/bars must NOT be present.
	for _, bad := range []string{"Made with", "gradbar", "Infinity"} {
		if strings.Contains(html, bad) {
			t.Errorf("rendered HTML unexpectedly contains %q", bad)
		}
	}
}

func TestComputedAmountAndTotal(t *testing.T) {
	inv := Invoice{
		Currency: "INR",
		Items: []LineItem{
			{Detail: "Contractor Hourly", RateValue: 1250.50, QtyValue: 72, RateUnit: "/hrs"},
		},
	}
	if _, err := inv.RenderHTML(); err != nil {
		t.Fatal(err)
	}
	// normalize runs inside RenderHTML on a copy; verify via direct call.
	inv.normalize()
	if inv.Items[0].Amount != "90,036.00" {
		t.Errorf("amount = %q, want 90,036.00", inv.Items[0].Amount)
	}
	if got := Money("INR", inv.Total); got != "90,036.00" {
		t.Errorf("total = %q, want 90,036.00", got)
	}
}

func TestTotalSumsRoundedAmounts(t *testing.T) {
	inv := Invoice{Currency: "USD", Items: []LineItem{
		{Detail: "a", RateValue: 19.99, QtyValue: 0.333},
		{Detail: "b", RateValue: 19.99, QtyValue: 0.333},
	}}
	inv.normalize()
	if inv.Items[0].Amount != "6.66" || Money("USD", inv.Total) != "13.32" {
		t.Errorf("amount = %s, total = %s; want 6.66, 13.32", inv.Items[0].Amount, Money("USD", inv.Total))
	}

	// Rate, amount and total use the same rounding: 2.675 is stored below 2.675.
	inv = Invoice{Currency: "USD", Items: []LineItem{{Detail: "c", RateValue: 2.675, QtyValue: 1}}}
	inv.normalize()
	if inv.Items[0].Rate != "2.67" || inv.Items[0].Amount != "2.67" || Money("USD", inv.Total) != "2.67" {
		t.Errorf("rate = %s, amount = %s, total = %s; want 2.67 each", inv.Items[0].Rate, inv.Items[0].Amount, Money("USD", inv.Total))
	}
}

func TestResolveAllRejectsBadInvoices(t *testing.T) {
	cases := map[string][]InvoiceSpec{
		"dots":      {{Number: "../important"}},
		"leading -": {{Number: "-19"}},
		"device":    {{Number: "nul"}},
		"com1":      {{Number: "COM1"}},
		"case dup":  {{Number: "INV-01"}, {Number: "inv-01"}},
		"huge rate": {{Number: "INV-1", Items: []LineItem{{RateValue: 1e308, QtyValue: 1}}}},
		"slash":     {{Number: "INV/1"}},
		"empty":     {{Number: ""}},
		"duplicate": {{Number: "INV-1"}, {Number: "INV-1"}},
		"inf total": {{Number: "INV-1", Total: math.Inf(1)}},
		"overflow":  {{Number: "INV-1", Items: []LineItem{{RateValue: 1e200, QtyValue: 1e200}}}},
	}
	for name, specs := range cases {
		if _, err := (&Config{Invoices: specs}).ResolveAll(); err == nil {
			t.Errorf("%s: ResolveAll accepted %+v", name, specs)
		}
	}
	if _, err := (&Config{Invoices: []InvoiceSpec{{Number: "INV-2026_01"}}}).ResolveAll(); err != nil {
		t.Errorf("valid number rejected: %v", err)
	}
}

func TestLoadConfigRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invoices.yaml")
	yml := "invoices:\n  - number: INV-1\n    items:\n      - detail: a\n        rateValue: 1000\n        qtyvalue: 8\n"
	if err := os.WriteFile(path, []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil || !strings.Contains(err.Error(), "qtyvalue") {
		t.Fatalf("LoadConfig error = %v, want unknown field qtyvalue", err)
	}
}

func TestLoadConfigRejectsSecondDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invoices.yaml")
	yml := "invoices:\n  - number: INV-1\n---\ninvoices:\n  - number: INV-2\n"
	if err := os.WriteFile(path, []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("LoadConfig accepted two YAML documents")
	}
}

func TestCheckNeedsTotalForLiteralAmounts(t *testing.T) {
	literal := Invoice{Number: "INV-1", Items: []LineItem{{Detail: "Fixed", Amount: "500.00"}, {Detail: "Hourly", RateValue: 10, QtyValue: 2}}}
	if err := literal.Check(); err == nil {
		t.Error("Check accepted literal amount without total")
	}
	literal.Total = 520
	if err := literal.Check(); err != nil {
		t.Errorf("Check rejected literal amount with total: %v", err)
	}
	mixed := Invoice{Number: "INV-3", Items: []LineItem{{Detail: "Changed", RateValue: 100, QtyValue: 2, Amount: "500.00"}}}
	if err := mixed.Check(); err == nil {
		t.Error("Check accepted literal amount with rateValue and no total")
	}
	calculated := Invoice{Number: "INV-2", Items: []LineItem{{Detail: "Hourly", RateValue: 10, QtyValue: 2}}}
	if err := calculated.Check(); err != nil {
		t.Errorf("Check rejected calculated items: %v", err)
	}
}

func TestRenderHTMLShowsBilledToDetails(t *testing.T) {
	inv := Invoice{Number: "INV-1", Currency: "INR", BilledTo: Party{Name: "Client", Email: "billing@example.com", TaxIDs: []string{"GSTIN: 29ABCDE1234F1Z5"}}}
	html, err := inv.RenderHTML()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"billing@example.com", "GSTIN: 29ABCDE1234F1Z5"} {
		if !strings.Contains(html, want) {
			t.Errorf("billed-to %q missing from HTML", want)
		}
	}
}

func TestFileURLEscapes(t *testing.T) {
	if got := fileURL("/tmp/inv#2026/a b.html"); got != "file:///tmp/inv%232026/a%20b.html" {
		t.Errorf("fileURL = %q", got)
	}
}

func TestRenderHTMLFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "invoice.html")
	if err := benchmarkInvoice().RenderHTMLFile(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "INV-0000-01") {
		t.Fatal("HTML file does not contain the invoice number")
	}
}

func BenchmarkMoneyINR(b *testing.B) {
	for b.Loop() {
		moneySink = Money("INR", 123456789.01)
	}
}

func BenchmarkMoneyUSD(b *testing.B) {
	for b.Loop() {
		moneySink = Money("USD", 123456789.01)
	}
}

func BenchmarkRenderHTML(b *testing.B) {
	inv := benchmarkInvoice()
	for b.Loop() {
		var err error
		htmlSink, err = inv.RenderHTML()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderHTMLToWriter(b *testing.B) {
	inv := benchmarkInvoice()
	for b.Loop() {
		if err := inv.renderHTML(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkInvoice() Invoice {
	return Invoice{
		Number:   "INV-0000-01",
		Date:     "09 Aug, 2026",
		Due:      "09 Aug, 2026",
		Currency: "INR",
		Seller: Party{
			Name:    "Jane Contractor",
			Email:   "jane@example.com",
			Address: []string{"123 Example Street"},
		},
		BilledTo: Party{
			Name:    "Acme Corp",
			Address: []string{"1 Market Plaza"},
		},
		Items: []LineItem{
			{Detail: "Consulting", Qty: "8", Rate: "1,000.00/hrs", Amount: "8,000.00"},
		},
		Payment: []PayRow{
			{Label: "Bank Name:", Value: "EXAMPLE BANK"},
		},
		Total: 8000,
	}
}

func BenchmarkResolveAll(b *testing.B) {
	cfg, err := LoadConfig("../../examples/invoices.example.yaml")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		invoicesSink, err = cfg.ResolveAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}
