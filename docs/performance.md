# Performance Guide

This project uses the GoPerf recommendations that apply to its workload.
Invoice PDF Creator is a command-line program that runs for a short time. It
processes a small and limited set of invoices.

## Run benchmarks

Run all benchmarks:

```sh
make bench
```

Run compiler escape analysis:

```sh
go build -gcflags="-m=2" ./...
```

Benchmark results depend on the processor, operating system, and Go version.
Compare results on the same machine.

## Applied recommendations

The code applies these recommendations:

- Put small scalar struct fields after fields that contain pointers.
- Preallocate slices, maps, strings, and render buffers when the size is known.
- Parse command values with string slices instead of temporary string arrays.
- Render HTML through a buffered writer when the output is a file.
- Parse the embedded template only when the program renders the first invoice.
- Do not change the parsed template after the program parses it.
- Examine the escape analysis output for new heap allocations.
- Use stripped release builds and remove local build paths.

The money formatter uses one allocation for its result. Invoice resolution uses
one allocation for the returned slice.

## Excluded recommendations

The code does not use `sync.Pool`. The program does not have a sustained stream
of short requests. A pool adds complexity and gives no clear benefit for the
normal command workload.

The code does not use worker pools, atomics, mutexes, or contexts. The program
makes a limited number of invoices, one after the other. PDF generation starts an external Chrome
process. Unmeasured parallel Chrome processes can increase memory use and make
performance less predictable.

The code does not change `GOGC` or `GOMEMLIMIT`. The process runs for a short
time. A profile does not show that garbage collection is a limit.

The code does not use memory mapping. Configuration files are small and are
read one time.

The network recommendations do not apply. The program does not provide a
network service.

Do not add one of these techniques without a benchmark or a production profile
that shows a benefit.
