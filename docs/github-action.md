# GitHub Action

```yaml
name: jvm-fit
on: pull_request
permissions:
  contents: read
jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: andronaft/jvm-fit@main   # pin a release tag once there is one
        with:
          path: k8s/
          fail-on: high
```

Each finding becomes an annotation on the manifest (or the Helm template)
in the pull request, the text report goes to the job summary, and the step
fails when a finding has the `fail-on` severity or higher. The action builds
jvm-fit from its own source, so it needs Go on the runner (it installs Go
if there is none).

| Input | Default | Meaning |
|---|---|---|
| `path` | `.` | files or directories, separated by spaces or new lines |
| `helm` | | charts to render (`repo/chart`, OCI, URL), one per line |
| `values` | | values files for the charts, one per line |
| `set` | | chart values, `key=value`, one per line |
| `fail-on` | `high` | as `--fail-on` |
| `nonheap-estimate` | `350Mi` | as `--nonheap-estimate` |
| `flags-env` | | as `--flags-env`, comma-separated |
| `sarif-file` | | also write a SARIF report to this file |
| `summary` | `true` | add the text report to the job summary |

Output: `findings`, the number of findings.

## Code scanning

Annotations are shown for files under the workspace; GitHub shows at most
10 errors and 10 warnings per step (50 per job). For a full list in the Security tab,
upload SARIF to code scanning
([example workflow](../.github/workflows/jvm-fit.yml)):

```yaml
permissions:
  contents: read
  security-events: write
steps:
  - uses: actions/checkout@v7
  - uses: andronaft/jvm-fit@main
    with:
      path: k8s/
      sarif-file: jvm-fit.sarif
  - uses: github/codeql-action/upload-sarif@v4
    if: always() && hashFiles('jvm-fit.sarif') != ''
    with:
      sarif_file: jvm-fit.sarif
      category: jvm-fit
```

Severities map to annotation and SARIF levels: critical and high → error,
medium → warning, low and info → notice (`note` in SARIF). The SARIF output
is validated against the official OASIS SARIF 2.1.0 schema in the tests.

Without the action, `--format github` prints the same annotations from any
workflow step, and `--format sarif` writes the SARIF file.
