# Invoice PDF Creator

Invoice PDF Creator is a command-line program for tax invoices. It reads invoice
data from a YAML configuration file (`invoices.yaml` by default). Command
options select invoices and override some fields. It writes an HTML file or a
PDF file.

The program uses an embedded HTML template. It uses Chrome or Chromium to make
a PDF file.

## Main functions

- Make one invoice or a set of invoices.
- Reuse seller, customer, tax, note, and bank data.
- Calculate line amounts from a rate and a quantity.
- Format INR amounts with Indian digit groups.
- Format other currencies with standard digit groups.
- Select invoices by invoice number.
- Write HTML without Chrome.

## Requirements

- Chrome or Chromium for PDF output. You do not need Chrome for HTML output.
- The Inter font for the specified document appearance.
- Go 1.26.4 or a later compatible version. You need Go only to build the
  program from the source code.

## Install

Download the binary for your system from the
[Releases](https://github.com/akhilnarang/invoice-pdf-creator/releases) page.

Or, build the program from the source code. Run this command in the repository
directory:

```sh
go build -o invoice-pdf-creator .
```

## Make your first invoice

First, copy the example configuration file:

```sh
cp examples/invoices.example.yaml invoices.yaml
```

Then, put your invoice data in `invoices.yaml`.

Do not put real invoice data in the example file. The repository ignores
`invoices.yaml`, but it tracks the example file.

Make HTML files first. Use the HTML files to check the data and the layout.

```sh
./invoice-pdf-creator gen \
  --config invoices.yaml \
  --out ./out \
  --html-only
```

Then, make the PDF files:

```sh
./invoice-pdf-creator gen \
  --config invoices.yaml \
  --out ./out
```

The program writes one file for each invoice. The default output directory is
`./out`.

## Select invoices

Use `--only` to select one or more invoices. The aliases are `--number` and
`--num`.

Give the full invoice number or only its last part. For example, `19` and `019`
select `INV-0002-19`.

You can repeat the option. You can also put the invoice numbers after the
options, without an option name. Put all options before the invoice numbers:

```sh
./invoice-pdf-creator gen --num 19 --num 20
./invoice-pdf-creator gen 19 20
./invoice-pdf-creator gen --out ./out 19 20
```

You can also give a list with commas:

```sh
./invoice-pdf-creator gen \
  --config invoices.yaml \
  --only INV-0000-01,INV-0000-03 \
  --out ./out
```

The program stops with an error in these conditions:

- A number is not in the configuration file.
- A short number agrees with more than one invoice.

## Command options

| Option | Function |
| --- | --- |
| `--config PATH` | Read invoice data from `PATH`. The default is `invoices.yaml`. |
| `--out DIR` | Write files to `DIR`. The default is `./out`. |
| `--only N[,N...]` | Make only the specified invoices. `--number` and `--num` are aliases. |
| `--html-only` | Write HTML files. Do not start Chrome. |
| `--chrome PATH` | Use the Chrome or Chromium program at `PATH`. |
| `--date TEXT` | Set the invoice date. |
| `--due TEXT` | Set the due date. |
| `--currency CODE` | Set the currency code. |
| `--total NUMBER` | Set the invoice total. Do not use zero. |
| `--note TEXT` | Set the footer note. |

Set `INVOICE_CHROME` to change the default Chrome command. The default command
is `google-chrome-stable`. The `--chrome` option has priority over the
environment variable.

These options replace values in all selected invoices:

- `--date`
- `--due`
- `--currency`
- `--note`
- `--total`

## Configuration

See [docs/configuration.md](docs/configuration.md) for the YAML fields, preset
rules, and amount rules.

The complete example is
[examples/invoices.example.yaml](examples/invoices.example.yaml). All identity
and bank values in this file are placeholders.

## Protect private data

Invoice configuration files and generated documents can contain personal data
and bank data. Keep this data out of Git.

Use the standard `invoices.yaml` name. Use the standard `out` directory. The
repository ignores these paths.

See [SECURITY.md](SECURITY.md) before you add a new configuration name or a new
output directory.

## Test the program

Run all tests:

```sh
go test ./...
```

The tests check the amount format, the calculated totals, the configuration
rules, and the HTML output.

Run a fuzz test for 60 seconds:

```sh
go test ./internal/invoice -run='^$' -fuzz='^FuzzConfigPipeline$' -fuzztime=60s
```

The other fuzz tests are `FuzzMoney` in `./internal/invoice`, and `FuzzParse`
and `FuzzSelectInvoices` in the repository root.

Run the benchmarks:

```sh
make bench
```

See [docs/performance.md](docs/performance.md) for the applied optimization
rules and the excluded GoPerf patterns.
