# jvm-fit

[![CI](https://github.com/andronaft/jvm-fit/actions/workflows/ci.yml/badge.svg)](https://github.com/andronaft/jvm-fit/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

**jvm-fit reads your pod spec and JVM flags together and tells you, before
deploy, if the JVM will fit in the container.**

Java in Kubernetes often breaks even when "everything is configured":

| What you see | The real cause |
|---|---|
| Pod is **OOMKilled**, heap is only 60% full | Heap takes almost the whole limit; **non-heap** (metaspace, thread stacks, code cache, direct buffers) does not fit |
| App is slow, CPU usage is low | **CPU limit** → throttling; with a limit below 2 cores the JVM picks **SerialGC** |
| Memory is "used", but the heap is small | No heap flags → heap is **25%** of the limit, the rest sits idle |
| Pod on a big node takes an 8Gi heap | **No memory limit** → the JVM takes a percentage of the **whole node** |
| Pod restarts during startup | Liveness probe kills a slow-starting JVM (no **startup probe**) |

Right-sizing tools such as KRR and Goldilocks recommend container resources
("give it 2Gi") but do not tell you that `-Xmx` must change too. jvm-fit
checks both together.

## Example

```
$ jvm-fit scan testdata/payments-api.yaml

payments-api (Deployment, container: app)  image: eclipse-temurin:17-jre
  testdata/payments-api.yaml:15  limits: memory=1Gi cpu=500m   flags: -Xmx900m (JAVA_TOOL_OPTIONS)

  Memory budget
    container limit        1024Mi
    heap max (-Xmx)         900Mi
    non-heap estimate      ~350Mi
    ─────────────────────────────
    needed                ~1250Mi   ❌ over limit by ~226Mi

  CPU   JVM sees 1 (CPU limit 500m)
  GC    Serial (CPU limit 500m → the JVM sees 1 CPU)

  HIGH     JVM003  Heap + non-heap needs ~1250Mi, limit is 1024Mi (over by ~226Mi) → OOMKilled under load
                   fix: -XX:MaxRAMPercentage=70 and memory limit 1.25Gi, or keep the limit and use -XX:MaxRAMPercentage=65
  MEDIUM   JVM007  CPU limit 500m → the JVM sees 1 CPU, no GC set → SerialGC (single-threaded GC)
                   fix: add -XX:+UseG1GC, or raise the limits to ≥ 2 CPUs and ≥ 1792Mi
  MEDIUM   JVM008  CPU limit 500m is below 1 core → throttling during startup, JIT and GC
                   fix: raise the CPU limit to ≥ 1, or keep only resources.requests.cpu
  MEDIUM   JVM012  No startup probe; liveness starts after 10s
                   fix: add a startupProbe (for example periodSeconds: 5, failureThreshold: 24 for up to 120s)
  INFO     JVM006  Heap is fixed with -Xmx900m (JAVA_TOOL_OPTIONS) and will not follow changes to the limit
                   fix: replace it with -XX:MaxRAMPercentage=70

5 findings (1 high, 3 medium, 1 info) in 1 of 1 workloads
```

## Install

```bash
go install github.com/andronaft/jvm-fit/cmd/jvm-fit@latest
```

Or build from source (`bin/jvm-fit`):

```bash
git clone https://github.com/andronaft/jvm-fit && cd jvm-fit && make build
```

## Usage

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

| Flag | Default | Meaning |
|---|---|---|
| `--format` | `text` | `text`, `json` or `sarif` (2.1.0) |
| `-o`, `--output` | stdout | write the report to a file |
| `--fail-on` | `high` | exit 1 when a finding has this severity or higher (`critical`, `high`, `medium`, `low`, `info`, `none`) |
| `--nonheap-estimate` | `350Mi` | memory the JVM needs outside the heap |
| `--flags-env` | | more env vars your entrypoint passes to `java` |
| `--helm` | | chart to render: a directory, `repo/chart`, an OCI reference or a `.tgz` URL (repeatable) |
| `-f`, `--values` | | values file for the charts given as arguments or with `--helm` (repeatable) |
| `--set` | | chart value, `key=value` (repeatable) |
| `--chart-version` | | chart version for `--helm` charts from a repository |

### Helm and Kustomize

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

Exit codes (the same for every format):

| Code | Meaning |
|---|---|
| `0` | no findings at or above `--fail-on` |
| `1` | at least one finding at or above `--fail-on` |
| `2` | error: bad flags, unreadable file, invalid YAML or a chart that fails to render |

### GitHub code scanning

```yaml
- run: jvm-fit scan k8s/ --format sarif -o jvm-fit.sarif --fail-on none
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: jvm-fit.sarif
    category: jvm-fit
```

Severities map to SARIF levels: critical and high → `error`, medium →
`warning`, low and info → `note`. The output is validated against the
official OASIS SARIF 2.1.0 schema in the tests.

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

## Rules

| ID | Checks | Severity |
|---|---|---|
| JVM001 | No memory limit, heap is a percentage (explicit or default) | high |
| JVM002 | `-Xmx` ≥ memory limit | critical |
| JVM003 | heap + non-heap estimate > memory limit | high |
| JVM004 | `MaxRAMPercentage` > 80 | medium |
| JVM005 | No `-Xmx` and no `MaxRAMPercentage` → heap is 25% of the limit (HotSpot) | low |
| JVM006 | Fixed `-Xmx` in absolute units | info |
| JVM007 | No GC set and the JVM will pick SerialGC (CPU < 2 or memory < 1792Mi) | medium |
| JVM008 | CPU limit < 1 core | medium |
| JVM009 | `-XX:-UseContainerSupport` | high |
| JVM010 | `ActiveProcessorCount` > CPU limit | medium |
| JVM011 | Memory request < limit | low |
| JVM012 | No startup probe and liveness `initialDelaySeconds` < 30 | medium |

Run `jvm-fit explain <ID>` for the full explanation.

## Evidence

The model is checked against real JVMs, not only against documentation:

- [`docs/evidence/results.md`](docs/evidence/results.md): what JDK 11, 17, 21
  (HotSpot) and OpenJ9 21 pick under different `--memory` / `--cpus` limits.
- `TestDockerEvidence` starts real JVMs in Docker with the same limits as the
  test cases and fails if the model predicts a different heap, CPU count or
  GC. It runs in CI on every push (`make evidence` locally).

Measured facts that are easy to get wrong:

- 2 CPUs with a 1Gi limit still gives **SerialGC** (G1 also needs ~1792Mi).
- A 256Mi limit gives a 126M heap, not 25% (64M).
- With `-XX:-UseContainerSupport` a 1Gi container gets a heap sized from the
  host memory.

See [`docs/evidence/README.md`](docs/evidence/README.md) for the setup and
what is not measured yet.

## Development

```bash
make test       # unit tests
make lint       # gofmt + go vet
make evidence   # model vs real JVMs in Docker
```

Rules live in `internal/rules`, one file per rule, each with a test that has
at least one good and one bad example.

## Roadmap

- GitHub Action with PR annotations
- Reading pods from a live cluster and real non-heap usage from Prometheus

## License

[Apache-2.0](LICENSE)
