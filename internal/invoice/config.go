package invoice

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level YAML document. It supports reusable presets so
// multiple invoices can reference banks and tax-id sets by name.
type Config struct {
	Defaults struct {
		Currency string `yaml:"currency"`
		Seller   Party  `yaml:"seller"`
		BilledTo Party  `yaml:"billedTo"`
	} `yaml:"defaults"`

	BankPresets  map[string][]PayRow `yaml:"bankPresets"`
	TaxIDPresets map[string][]string `yaml:"taxIdPresets"`
	Notes        map[string]string   `yaml:"notes"`

	Invoices []InvoiceSpec `yaml:"invoices"`
}

// InvoiceSpec is an invoice entry in the config; preset names are resolved
// against the Config's presets when building the concrete Invoice.
type InvoiceSpec struct {
	Number   string     `yaml:"number"`
	Date     string     `yaml:"date"`
	Due      string     `yaml:"due"`
	Currency string     `yaml:"currency,omitempty"`
	Seller   *Party     `yaml:"seller,omitempty"`
	BilledTo *Party     `yaml:"billedTo,omitempty"`
	Items    []LineItem `yaml:"items"`

	Bank     string   `yaml:"bank,omitempty"`
	Payment  []PayRow `yaml:"payment,omitempty"`
	TaxIDs   string   `yaml:"taxIds,omitempty"`
	TaxList  []string `yaml:"taxList,omitempty"`
	Note     string   `yaml:"note,omitempty"`
	NoteName string   `yaml:"noteName,omitempty"`
	Total    float64  `yaml:"total,omitempty"`
}

// LoadConfig reads and parses a YAML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	// Reject unknown fields: a typo like "qtyvalue" would otherwise bill 0.00.
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	// A second "---" document would otherwise be skipped without a word.
	if err := dec.Decode(new(yaml.Node)); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("cannot read %s: the file must contain only one YAML document", path)
	}
	return &c, nil
}

// Resolve turns a spec into a concrete Invoice, applying defaults and presets.
func (c *Config) Resolve(s InvoiceSpec) (Invoice, error) {
	inv := Invoice{
		Number:   s.Number,
		Date:     s.Date,
		Due:      s.Due,
		Currency: firstNonEmpty(s.Currency, c.Defaults.Currency),
		Items:    s.Items,
		Total:    s.Total,
		Note:     s.Note,
	}

	if s.Seller != nil {
		inv.Seller = *s.Seller
	} else {
		inv.Seller = c.Defaults.Seller
	}
	if s.BilledTo != nil {
		inv.BilledTo = *s.BilledTo
	} else {
		inv.BilledTo = c.Defaults.BilledTo
	}

	// Tax IDs: inline list wins, else preset name.
	if len(s.TaxList) > 0 {
		inv.Seller.TaxIDs = s.TaxList
	} else if s.TaxIDs != "" {
		ids, ok := c.TaxIDPresets[s.TaxIDs]
		if !ok {
			return Invoice{}, fmt.Errorf("invoice %s: the taxIds preset %q does not exist", s.Number, s.TaxIDs)
		}
		inv.Seller.TaxIDs = ids
	}

	// Payment: inline wins, else bank preset.
	if len(s.Payment) > 0 {
		inv.Payment = s.Payment
	} else if s.Bank != "" {
		p, ok := c.BankPresets[s.Bank]
		if !ok {
			return Invoice{}, fmt.Errorf("invoice %s: the bank preset %q does not exist", s.Number, s.Bank)
		}
		inv.Payment = p
	}

	// Note preset (literal Note already set above takes precedence).
	if inv.Note == "" && s.NoteName != "" {
		n, ok := c.Notes[s.NoteName]
		if !ok {
			return Invoice{}, fmt.Errorf("invoice %s: the note %q does not exist", s.Number, s.NoteName)
		}
		inv.Note = n
	}

	return inv, nil
}

// validNumber keeps invoice numbers usable as file names inside the output
// directory: no dots, slashes, or spaces, and no leading "-" (it reads as a flag).
var validNumber = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// windowsDevice matches names Windows maps to devices, even with an extension.
var windowsDevice = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[0-9]|LPT[0-9])$`)

// maxAmount bounds every amount so rounding and formatting stay finite.
const maxAmount = 1e15

// ResolveAll resolves every invoice in the config and rejects invalid or
// duplicate invoice numbers and non-finite amounts.
func (c *Config) ResolveAll() ([]Invoice, error) {
	out := make([]Invoice, len(c.Invoices))
	seen := make(map[string]bool, len(c.Invoices))
	for i, s := range c.Invoices {
		if !validNumber.MatchString(s.Number) || windowsDevice.MatchString(s.Number) {
			return nil, fmt.Errorf("invoice %d: the number %q is not correct (use only letters, digits, - and _; start with a letter or a digit; do not use a Windows device name)", i+1, s.Number)
		}
		// Case-insensitive, since macOS and Windows file names usually are.
		key := strings.ToLower(s.Number)
		if seen[key] {
			return nil, fmt.Errorf("invoice %s: another invoice has the same number (the program does not compare letter case)", s.Number)
		}
		seen[key] = true
		if !validAmount(s.Total) {
			return nil, fmt.Errorf("invoice %s: the total must be a number less than %g", s.Number, maxAmount)
		}
		for _, it := range s.Items {
			if !validAmount(it.RateValue) || !validAmount(it.RateValue*it.QtyValue) {
				return nil, fmt.Errorf("invoice %s: item %q: rateValue and the calculated amount must be numbers less than %g", s.Number, it.Detail, maxAmount)
			}
		}
		inv, err := c.Resolve(s)
		if err != nil {
			return nil, err
		}
		out[i] = inv
	}
	return out, nil
}

// validAmount is false for NaN and ±Inf too, since those compare false.
func validAmount(v float64) bool { return math.Abs(v) < maxAmount }

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
