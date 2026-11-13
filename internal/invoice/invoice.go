// Package invoice models tax invoices and renders them to HTML/PDF using an
// embedded template that matches the established design.
package invoice

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
)

//go:embed template.html
var templateHTML string

var invoiceTemplate = sync.OnceValue(func() *template.Template {
	return template.Must(template.New("invoice").Parse(templateHTML))
})

// Party is the seller or the bill-to entity.
type Party struct {
	Name    string   `yaml:"name"`
	Email   string   `yaml:"email,omitempty"`
	Address []string `yaml:"address"`
	TaxIDs  []string `yaml:"taxIds,omitempty"`
}

// LineItem is a single row of the invoice table. Rate/Amount may be supplied
// as literal display strings, or computed from RateValue*Qty when RateValue>0.
type LineItem struct {
	Detail   string `yaml:"detail"`
	HSN      string `yaml:"hsn,omitempty"`
	Qty      string `yaml:"qty"`
	Rate     string `yaml:"rate"`
	Amount   string `yaml:"amount,omitempty"`
	RateUnit string `yaml:"rateUnit,omitempty"`

	// Optional computed inputs. When RateValue != 0 and Amount is empty,
	// Amount is computed as RateValue * QtyValue and formatted for Currency.
	RateValue float64 `yaml:"rateValue,omitempty"`
	QtyValue  float64 `yaml:"qtyValue,omitempty"`
}

// PayRow is one label/value line in the Payment Instructions block.
type PayRow struct {
	Label string `yaml:"label"`
	Value string `yaml:"value"`
}

// Invoice is one renderable invoice.
type Invoice struct {
	Number   string     `yaml:"number"`
	Date     string     `yaml:"date"`
	Due      string     `yaml:"due"`
	Currency string     `yaml:"currency"`
	Seller   Party      `yaml:"seller"`
	BilledTo Party      `yaml:"billedTo"`
	Items    []LineItem `yaml:"items"`
	Payment  []PayRow   `yaml:"payment"`
	Note     string     `yaml:"note,omitempty"`
	Total    float64    `yaml:"total,omitempty"`
}

// view is the data actually handed to the template (with derived fields).
type view struct {
	Invoice
	TotalFmt string
	NBSP     template.HTML
}

// Money formats v with currency-appropriate digit grouping and 2 decimals.
// INR uses Indian grouping (2,46,810.00); everything else uses standard
// 3-digit grouping (3,141.50).
func Money(currency string, v float64) string {
	neg := math.Signbit(v)
	// Let strconv round to 2 dp using the correct rounding of the actual
	// float value (no hand-rolled epsilon, no double-rounding). Then group the
	// integer-part digits. Note: because float64 cannot represent every decimal
	// exactly, a decimal "midpoint" literal like 1.005 is actually stored just
	// below 1.005 and rounds to 1.00 — that is the correct rounding of the
	// stored value. Pass already-rounded values (or integer cents) if exact
	// half-up behaviour matters.
	s := strconv.FormatFloat(math.Abs(v), 'f', 2, 64)
	dot := strings.IndexByte(s, '.')
	intPart, frac := s[:dot], s[dot+1:]

	isINR := strings.EqualFold(currency, "INR")
	commas := (len(intPart) - 1) / 3
	if isINR {
		rest := len(intPart) - 3
		if rest > 0 {
			commas = (rest + 1) / 2
		}
	}
	isNegative := neg && s != "0.00"
	size := len(s) + commas
	if isNegative {
		size++
	}
	var b strings.Builder
	b.Grow(size)
	if isNegative {
		b.WriteByte('-')
	}
	if isINR {
		writeIndian(&b, intPart)
	} else {
		writeStandard(&b, intPart)
	}
	b.WriteByte('.')
	b.WriteString(frac)
	return b.String()
}

// round2 rounds v to 2 decimals exactly the way Money displays it.
func round2(v float64) float64 {
	r, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'f', 2, 64), 64)
	return r
}

func writeIndian(b *strings.Builder, d string) {
	if len(d) <= 3 {
		b.WriteString(d)
		return
	}
	rest := len(d) - 3
	first := rest % 2
	if first == 0 {
		first = 2
	}
	b.WriteString(d[:first])
	for i := first; i < rest; i += 2 {
		b.WriteByte(',')
		b.WriteString(d[i : i+2])
	}
	b.WriteByte(',')
	b.WriteString(d[rest:])
}

func writeStandard(b *strings.Builder, d string) {
	if len(d) <= 3 {
		b.WriteString(d)
		return
	}
	first := len(d) % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(d[:first])
	for i := first; i < len(d); i += 3 {
		b.WriteByte(',')
		b.WriteString(d[i : i+3])
	}
}

// Check reports problems that would render a wrong invoice. Run it after
// overrides and before writing any file.
func (inv Invoice) Check() error {
	if inv.Total != 0 {
		return nil
	}
	for _, it := range inv.Items {
		if it.Amount != "" { // literal amounts are never summed, even with rateValue
			return fmt.Errorf("invoice %s: item %q has a literal amount, so you must set the total", inv.Number, it.Detail)
		}
	}
	return nil
}

// normalize fills computed amounts and the invoice total.
func (inv *Invoice) normalize() {
	var sum float64
	for i := range inv.Items {
		it := &inv.Items[i]
		if it.RateValue != 0 {
			if it.Rate == "" {
				it.Rate = Money(inv.Currency, it.RateValue) + it.RateUnit
			}
			// Round each row like Money does, so the total equals the sum of the
			// displayed amounts.
			amt := round2(it.RateValue * it.QtyValue)
			if it.Amount == "" {
				it.Amount = Money(inv.Currency, amt)
			}
			sum += amt
		}
	}
	if inv.Total == 0 {
		inv.Total = sum
	}
}

// RenderHTML returns the invoice as an HTML document.
func (inv Invoice) RenderHTML() (string, error) {
	var buf bytes.Buffer
	buf.Grow(len(templateHTML))
	if err := inv.renderHTML(&buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (inv Invoice) renderHTML(w io.Writer) error {
	inv.normalize()
	v := view{
		Invoice:  inv,
		TotalFmt: Money(inv.Currency, inv.Total),
		NBSP:     template.HTML("&nbsp;"),
	}
	return invoiceTemplate().Execute(w, v)
}
