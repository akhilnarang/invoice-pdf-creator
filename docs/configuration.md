# Configuration Reference

Use a YAML configuration file to make complete invoices. The file can contain
defaults, presets, and one or more invoices.

Start with the example file:

```sh
cp examples/invoices.example.yaml invoices.yaml
```

The repository ignores `invoices.yaml`. Do not put real data in the example
file.

## Document structure

```yaml
defaults:
  currency: INR
  seller:
    name: Example Seller
    email: seller@example.com
    address:
      - 1 Example Road
  billedTo:
    name: Example Customer
    address:
      - 2 Example Road

taxIdPresets:
  standard:
    - "PAN: ABCDE1234F"

notes:
  standard: "Example note."

bankPresets:
  standard:
    - { label: "Bank Name:", value: "EXAMPLE BANK" }
    - { label: "Account Number:", value: "00000000000" }

invoices:
  - number: INV-0000-01
    date: "09 Aug, 2026"
    due: "09 Aug, 2026"
    bank: standard
    taxIds: standard
    noteName: standard
    items:
      - detail: Consulting
        qty: "8"
        rateValue: 1000.00
        qtyValue: 8
        rateUnit: /hrs
```

## Defaults

Use `defaults` for data that applies to more than one invoice.

| Field | Function |
| --- | --- |
| `currency` | Sets the default currency code. |
| `seller` | Sets the default seller. |
| `billedTo` | Sets the default customer. |

An invoice can contain its own `seller`, `billedTo`, or `currency` value. An
invoice value has priority over a default value.

A party can contain these fields:

| Field | Function |
| --- | --- |
| `name` | Sets the party name. |
| `email` | Sets the email address. This field is optional. |
| `address` | Sets the address lines. |
| `taxIds` | Sets the tax identifier lines. This field is optional. |

## Presets

Presets prevent repeated data.

| Section | Function |
| --- | --- |
| `bankPresets` | Stores named lists of payment rows. |
| `taxIdPresets` | Stores named lists of tax identifier lines. |
| `notes` | Stores named footer notes. |

Use `bank`, `taxIds`, and `noteName` in an invoice to select presets.

You can put the data directly in an invoice. Use `payment` instead of `bank`.
Use `taxList` instead of `taxIds`. Use `note` instead of `noteName`.

Direct `payment` and `taxList` data have priority over presets. A direct `note`
has priority over a note preset.

The program stops if an invoice refers to a preset that does not exist.

## Invoice fields

| Field | Function |
| --- | --- |
| `number` | Sets the invoice number and the output file name. |
| `date` | Sets the invoice date text. |
| `due` | Sets the due date text. |
| `currency` | Replaces the default currency. |
| `seller` | Replaces the default seller. |
| `billedTo` | Replaces the default customer. |
| `items` | Sets the line items. |
| `total` | Sets the total. This field is optional for calculated items. |
| `bank` | Selects a bank preset. |
| `payment` | Sets payment rows directly. |
| `taxIds` | Selects a tax identifier preset. |
| `taxList` | Sets tax identifier lines directly. |
| `noteName` | Selects a note preset. |
| `note` | Sets the footer note directly. |

Obey these rules for invoice numbers:

- Use only letters, digits, `-`, and `_`.
- Start the number with a letter or a digit.
- Do not use a Windows device name, for example `CON` or `NUL`.
- Give each invoice a different number. The program does not compare letter
  case, so `INV-01` and `inv-01` are the same number.

Each total, rate, and calculated amount must be less than 10^15.

If the configuration does not obey these rules, the program stops with an
error. It stops before it writes a file.

## Calculated line items

Use these fields when the program must calculate an amount:

```yaml
- detail: Consulting
  hsn: "998311"
  qty: "8"
  rateValue: 1000.00
  qtyValue: 8
  rateUnit: /hrs
```

The program calculates `rateValue * qtyValue`. It formats the result and puts
it in `amount`. It also makes the displayed rate from `rateValue` and
`rateUnit` when `rate` is empty.

The program rounds each calculated amount to two decimal places. When `total`
is zero, it adds these rounded amounts.

## Literal line items

Use these fields when you must control the displayed text:

```yaml
- detail: Fixed service
  qty: "1"
  rate: "500.00/unit"
  amount: "500.00"
```

The program does not read or add a literal `amount` value. Thus, you must set
the total when you use literal amounts. Use the `total` field or the `--total`
option. If an invoice has a literal amount and no total, the program stops with
an error.

## File rules

The configuration file must contain only one YAML document.

Use only the field names in this reference. If the program finds an unknown
field name, it stops with an error. Thus, an incorrectly spelled field name
cannot cause an incorrect amount.

## Amount format

The program always shows two decimal places.

For `INR`, it uses Indian digit groups. For example, it shows `246810` as
`2,46,810.00`.

For all other currency codes, it uses groups of three digits. For example, it
shows `3141.5` as `3,141.50`.

The program uses binary floating-point numbers for calculations. If you must
have an exact decimal result, calculate the final total before you run the
program. Then, set `total` in the invoice.
