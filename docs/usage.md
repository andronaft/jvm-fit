# Usage

```bash
jvm-fit scan ./k8s/                        # files and directories (recursive)
jvm-fit scan ./charts/payments -f values-prod.yaml   # Helm chart (helm template)
jvm-fit scan --helm bitnami/tomcat --chart-version 11.3.1   # chart from a repo or OCI
jvm-fit scan ./k8s/overlays/prod           # Kustomize (kustomize build)
helm template my-chart | jvm-fit scan -    # stdin
jvm-fit scan ./k8s/ --format json          # machine-readable
jvm-fit scan ./k8s/ --format sarif -o jvm-fit.sarif   # for GitHub code scanning
jvm-fit explain                            # list rules
jvm-fit explain JVM007                     # why it is a problem and how to fix it
```

To build from source (`bin/jvm-fit`):

```bash
git clone https://github.com/andronaft/jvm-fit && cd jvm-fit && make build
```

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `--format` | `text` | `text`, `json`, `sarif` (2.1.0) or `github` (GitHub Actions annotations) |
| `-o`, `--output` | stdout | write the report to a file |
| `--fail-on` | `high` | exit 1 when a finding has this severity or higher (`critical`, `high`, `medium`, `low`, `info`, `none`) |
| `--nonheap-estimate` | `350Mi` | memory the JVM needs outside the heap |
| `--flags-env` | | more env vars your entrypoint passes to `java` |
| `--helm` | | chart to render: a directory, `repo/chart`, an OCI reference or a `.tgz` URL (repeatable) |
| `-f`, `--values` | | values file for the charts given as arguments or with `--helm` (repeatable) |
| `--set` | | chart value, `key=value` (repeatable) |
| `--chart-version` | | chart version for `--helm` charts from a repository |

## Exit codes

The same for every format:

| Code | Meaning |
|---|---|
| `0` | no findings at or above `--fail-on` |
| `1` | at least one finding at or above `--fail-on` |
| `2` | error: bad flags, unreadable file, invalid YAML or a chart that fails to render |

## Helm and Kustomize

- A directory with `Chart.yaml` is rendered with `helm template`; with
  `kustomization.yaml`, with `kustomize build` (or `kubectl kustomize` when
  `kustomize` is not installed). `helm` / `kubectl` must be in `PATH`.
- Findings point at the template file (`charts/payments/templates/deployment.yaml`)
  or the `kustomization.yaml`, without a line number: the rendered output
  has no line numbers of the sources.
- A chart found while walking a directory is rendered with its default
  values; if that fails, it is skipped with a warning. Kustomizations found
  while walking are not built (a base and its overlays would be reported
  twice), so pass the overlay directory itself.

## How it works

1. **Load manifests**: YAML files, Helm charts (`helm template`) and
   Kustomize overlays (`kustomize build`).
2. **Find Java containers**: JDK/JRE images (Temurin, Corretto, Semeru, Zulu,
   Liberica, distroless/java, Tomcat...), a `java` command, or `JAVA_*` env vars.
3. **Collect JVM flags** in the order the JVM applies them (later wins):
   `JAVA_TOOL_OPTIONS` → `JDK_JAVA_OPTIONS` → `JAVA_OPTS`-style vars →
   `command`/`args`. Kubernetes `$(VAR)` references and
   `sh -c "java $JAVA_OPTS ..."` are expanded. `JAVA_OPTS` works only if the
   entrypoint passes it to `java`, so jvm-fit marks those flags as **assumed**,
   unless the command uses the var directly. Vars from a ConfigMap or Secret
   are reported as not checked.
4. **Model how the JVM sees the container**:
   - max heap from `-Xmx`, `MaxRAMPercentage` or the JVM default (HotSpot: 25%
     of the limit, with a 126Mi floor for small limits; OpenJ9: 50% below 1Gi);
   - CPU count from the CPU limit (rounded up) or `ActiveProcessorCount`;
   - the GC the JVM picks: SerialGC unless it sees 2+ CPUs **and** ~1792Mi+;
   - memory budget: `heap max + non-heap estimate ≤ limit`.
5. **Run the rules** and print findings with a fix.

The non-heap number is an **estimate** (default 350Mi, typical for Spring
Boot). Change it with `--nonheap-estimate` if you know your app.

## Development

```bash
make test       # unit tests
make lint       # gofmt + go vet
make evidence   # model vs real JVMs in Docker
make demo       # OOMKill and fix on a kind cluster
```

Rules live in `internal/rules`, one file per rule, each with a test that has
at least one good and one bad example.
