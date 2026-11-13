// invoice-pdf-creator is a tiny CLI that renders Tax Invoice PDFs from an
// embedded HTML template, driven by a YAML config file.
//
// Usage:
//
//	invoice-pdf-creator gen [--config invoices.yaml] [--only INV-0002-03] [--out ./out] [--html-only]
//	invoice-pdf-creator version
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"invoice-pdf-creator/internal/invoice"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "gen":
		if err := runGen(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		_, _ = os.Stdout.WriteString("invoice-pdf-creator " + version + "\n")
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	_, _ = os.Stderr.WriteString(`invoice-pdf-creator makes tax invoice PDF files from a YAML configuration file.

Commands:
  gen        Make invoices from the configuration file.
  version    Show the version.
  help       Show this help.

Run "invoice-pdf-creator gen --help" to see the gen options.
`)
}

func runGen(args []string) error {
	opts := newOptions()
	if err := opts.parse(args); err != nil {
		return err
	}
	if opts.help {
		genUsage()
		return nil
	}

	if opts.config == "" {
		return fmt.Errorf("--config must not be empty")
	}
	cfg, err := invoice.LoadConfig(opts.config)
	if err != nil {
		return err
	}
	invoices, err := cfg.ResolveAll()
	if err != nil {
		return err
	}

	if len(opts.only) > 0 {
		if invoices, err = selectInvoices(invoices, opts.only); err != nil {
			return fmt.Errorf("%w in %s", err, opts.config)
		}
	}

	// Flag overrides applied to every selected invoice (handy with --only).
	for i := range invoices {
		opts.applyOverrides(&invoices[i])
		if err := invoices[i].Check(); err != nil {
			return err
		}
	}

	if len(invoices) == 0 {
		return fmt.Errorf("there are no invoices in %s", opts.config)
	}

	for _, inv := range invoices {
		if opts.htmlOnly {
			path := filepath.Join(opts.out, inv.Number+".html")
			if err := inv.RenderHTMLFile(path); err != nil {
				return err
			}
			fmt.Println("wrote", path)
			continue
		}
		path := filepath.Join(opts.out, inv.Number+".pdf")
		if err := inv.RenderPDF(path, opts.chrome); err != nil {
			return err
		}
		fmt.Println("wrote", path)
	}
	return nil
}

// ---- flags ----

type options struct {
	config, out, chrome       string
	date, due, currency, note string
	only                      []string
	total                     float64
	htmlOnly                  bool
	help                      bool
}

func newOptions() options {
	return options{config: "invoices.yaml", out: "./out", chrome: chromeDefault()}
}

func chromeDefault() string {
	if v := os.Getenv("INVOICE_CHROME"); v != "" {
		return v
	}
	return "google-chrome-stable"
}

func genUsage() {
	_, _ = os.Stderr.WriteString(`gen makes invoices from the configuration file.

Usage: invoice-pdf-creator gen [options] [N ...]

You can also put invoice numbers after the options: gen 19 20.
Put all options before the invoice numbers.

  --config PATH       Read invoice data from PATH. The default is invoices.yaml.
                      See examples/invoices.example.yaml.
  --only N[,N...]     Make only these invoices. The aliases are --number and --num.
                      You can repeat the option.
                      N is a full number or its last part. 19 and 019 select INV-0002-19.
  --out DIR           Write files to DIR. The default is ./out.
  --html-only         Write HTML files. Do not start Chrome.
  --chrome PATH       Use the Chrome or Chromium program at PATH. The default is
                      $INVOICE_CHROME or google-chrome-stable.

These options replace values in all selected invoices:
  --date TEXT         Set the invoice date. For example: "05 Jan, 2026".
  --due TEXT          Set the due date.
  --currency CODE     Set the currency code.
  --total NUMBER      Set the invoice total. Do not use zero.
  --note TEXT         Set the footer note. For example: an LUT declaration.
`)
}

func (o *options) parse(args []string) error {
	fs := flag.NewFlagSet("gen", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // genUsage prints the help; errors are returned
	fs.StringVar(&o.config, "config", o.config, "")
	fs.StringVar(&o.out, "out", o.out, "")
	fs.StringVar(&o.chrome, "chrome", o.chrome, "")
	fs.BoolVar(&o.htmlOnly, "html-only", false, "")
	fs.StringVar(&o.date, "date", "", "")
	fs.StringVar(&o.due, "due", "", "")
	fs.StringVar(&o.currency, "currency", "", "")
	fs.StringVar(&o.note, "note", "", "")
	for _, name := range []string{"only", "number", "num"} {
		fs.Func(name, "", o.addNumbers)
	}
	fs.Func("total", "", func(v string) error {
		t, err := strconv.ParseFloat(v, 64)
		// 0 means "sum the items", so it can't be an override.
		if err != nil || t == 0 || math.IsNaN(t) || math.IsInf(t, 0) {
			return errors.New("it must be a number, and it must not be zero")
		}
		o.total = t
		return nil
	})
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			o.help = true
			return nil
		}
		return err
	}
	// Arguments after the options are invoice numbers too: gen 19 20.
	for _, a := range fs.Args() {
		if err := o.addNumbers(a); err != nil {
			return fmt.Errorf("%q: %w (put all options before the invoice numbers)", a, err)
		}
	}
	return nil
}

// addNumbers adds a comma list of invoice numbers to the selection.
func (o *options) addNumbers(v string) error {
	ns := splitCSV(v)
	if len(ns) == 0 {
		return errors.New("it must have one or more invoice numbers")
	}
	for _, n := range ns {
		if strings.HasPrefix(n, "-") {
			return errors.New("an invoice number must not start with -")
		}
	}
	o.only = append(o.only, ns...)
	return nil
}

func (o *options) applyOverrides(inv *invoice.Invoice) {
	if o.date != "" {
		inv.Date = o.date
	}
	if o.due != "" {
		inv.Due = o.due
	}
	if o.currency != "" {
		inv.Currency = o.currency
	}
	if o.note != "" {
		inv.Note = o.note
	}
	if o.total != 0 {
		inv.Total = o.total
	}
}

// selectInvoices returns the invoices named by wants, in config order. A want
// matches an exact number, or else the number's last "-" part ignoring leading zeros.
func selectInvoices(invoices []invoice.Invoice, wants []string) ([]invoice.Invoice, error) {
	picked := make(map[int]bool, len(wants))
	for _, w := range wants {
		var exact, suffix []int
		for i, inv := range invoices {
			if inv.Number == w {
				exact = append(exact, i)
			} else if sameSuffix(inv.Number, w) {
				suffix = append(suffix, i)
			}
		}
		match := exact
		if len(match) == 0 {
			match = suffix
		}
		switch len(match) {
		case 0:
			return nil, fmt.Errorf("there is no invoice %s", w)
		case 1:
			picked[match[0]] = true
		default:
			names := make([]string, len(match))
			for j, i := range match {
				names[j] = invoices[i].Number
			}
			return nil, fmt.Errorf("invoice %s agrees with more than one invoice (%s)", w, strings.Join(names, ", "))
		}
	}
	out := make([]invoice.Invoice, 0, len(picked))
	for i, inv := range invoices {
		if picked[i] {
			out = append(out, inv)
		}
	}
	return out, nil
}

func sameSuffix(number, want string) bool {
	last := number[strings.LastIndex(number, "-")+1:]
	trim := func(s string) string {
		if t := strings.TrimLeft(s, "0"); t != "" || s == "" {
			return t
		}
		return "0"
	}
	return trim(last) == trim(want)
}

func splitCSV(s string) []string {
	out := make([]string, 0, strings.Count(s, ",")+1)
	for {
		p, rest, more := strings.Cut(s, ",")
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
		if !more {
			return out
		}
		s = rest
	}
}
