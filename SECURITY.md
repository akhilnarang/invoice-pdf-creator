# Private Data and Secrets

Invoice files can contain personal data, tax identifiers, addresses, and bank
data. Generated HTML and PDF files can contain the same data.

## Use the protected paths

Use `invoices.yaml` for real invoice data. Use `out` for generated files. The
repository ignores these paths.

The ignore rules also protect these root-level configuration names:

- `invoices*.yaml`
- `invoices*.yml`

The repository ignores PDF files in all directories. It does not ignore HTML
files in all directories. Put generated HTML files in `out`.

## Check a path

Use this command before you put private data in a file:

```sh
git check-ignore -v invoices.yaml
```

Git must show the rule that applies to the file. If Git shows no rule, do not put private data
in the file.

## Check a commit

Run these commands before each commit:

```sh
git status --short --ignored
git diff --cached
```

Read the complete staged change. Look for names, email addresses, postal
addresses, tax identifiers, account numbers, tokens, and private keys.

An ignore rule does not protect a file that Git already tracks. If Git tracks a
private file, remove it from the Git index before you make another commit:

```sh
git rm --cached -- invoices.yaml
```

This command keeps the local file. Confirm the staged change before you commit.

## Respond to an exposure

If a commit contains a secret, revoke or replace the secret immediately. A
later commit does not remove the secret from Git history.

If a commit contains personal data, stop the distribution of the repository.
Then, remove the data from Git history with an approved procedure. Tell all
repository users before you change the history.

Commit metadata can also contain a personal name and email address. Set a Git
author identity that is suitable for the repository before you make a commit.
