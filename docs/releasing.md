# Releasing

This guide is for maintainers.

## Make a GitHub release

Push a version tag to start the release workflow:

```sh
git tag v0.1.0
git push origin v0.1.0
```

You can also start the workflow manually. Open the Actions page. Select the
Release workflow. Enter a tag that exists, for example `v0.1.0`. The workflow
builds the commit of that tag.

The workflow tests the program and makes these files:

- `invoice-pdf-creator_<version>_windows_amd64.exe`
- `invoice-pdf-creator_<version>_linux_amd64`
- `invoice-pdf-creator_<version>_linux_arm64`
- `invoice-pdf-creator_<version>_darwin_arm64`

The workflow adds the binaries and a `SHA256SUMS` file to the GitHub release.

## Make a release binary on your computer

Use this command for a smaller binary:

```sh
make release VERSION=0.1.0
```

This build removes debug tables and local file-system paths. It also sets the
version string at build time.
