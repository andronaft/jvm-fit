# Evidence

Every number the model uses is checked on a real JVM. There are two parts:

1. **`results.md`**: a table made by [`run.sh`](run.sh). It starts JDK 11, 17
   and 21 (Eclipse Temurin, HotSpot) and OpenJ9 21 (IBM Semeru) in Docker with
   different `--memory` and `--cpus` values and records the max heap, the CPU
   count and the GC that the JVM picks.
2. **`TestDockerEvidence`** in
   [`internal/model/docker_evidence_test.go`](../../internal/model/docker_evidence_test.go):
   builds the model for a container spec, starts the same JVM in Docker with
   the same limits and flags, and fails if heap (±2%), CPU count or GC differ.
   It runs in CI on every push (`evidence` job) and locally with
   `make evidence`.

Run them yourself:

```bash
make evidence          # model vs real JVM
make evidence-table    # regenerate results.md
```

## Environment of `results.md`

Docker 20.10.8 on an arm64 Mac, **cgroup v1**, Docker VM with 4 CPUs and
**1988MiB** memory, 2026-11-02.

> ⚠️ Rows with `--memory=2g` or `4g` are above the Docker VM memory, so the
> JVM saw 1988MiB, not the limit. Those rows do **not** show limit behaviour
> (all of them show 498M = 25% of 1988MiB). Locally `TestDockerEvidence` skips
> these cases; the CI runner (more memory, cgroup v2) runs them.

## What the measurements show

HotSpot, the same on JDK 11, 17 and 21:

| Finding | Rows | Used by |
|---|---|---|
| Without heap flags the max heap is **25% of the limit**: 512m → 128M, 1g → 256M | 512m, 1g | model, JVM005 |
| Small limits use a floor instead: 256m → **126M** (not 64M) | 256m | model |
| `MaxRAMPercentage=75` with 1g → 768M | 1g + flag | model, JVM003/JVM004 |
| The JVM CPU count is the CPU limit rounded up: 0.5 → 1, 1 → 1, 2 → 2 | all | model, JVM007 |
| **1 CPU → SerialGC**, whatever the memory | 0.5, 1 | JVM007 |
| **2 CPUs but 1g memory → still SerialGC** (G1 needs 2+ CPUs *and* ~1792MiB) | 1g / 2 | JVM007 |
| `ActiveProcessorCount=2` with 1g → still SerialGC; with ~2g → G1 | flag rows | JVM007, JVM010 |
| `-XX:-UseContainerSupport` with a 1g limit → heap 498M (25% of the *host*), G1 | flag rows | JVM009 |

OpenJ9 21 (Semeru):

| Finding | Rows |
|---|---|
| Without heap flags the max heap is **50% of the limit**: 256m → 128M, 512m → 256M, 1g → 512M | 256m–1g |
| `MaxRAMPercentage=75` works the same as in HotSpot: 1g → 768M | 1g + flag |
| CPU count follows the CPU limit like HotSpot | all |

## TODO

- [ ] OpenJ9 above 1Gi: the model uses the documented defaults (limit − 512Mi
      for 1–2Gi, 75% above 2Gi). Not measured yet: needs a Docker host with
      more than 4GiB of memory.
- [ ] HotSpot G1 selection at exactly 2Gi / 2 CPUs on a host with enough
      memory (the local rows were capped by the VM; CI covers it in
      `TestDockerEvidence`, but `results.md` should show it too).
- [ ] Rerun `run.sh` on cgroup v2 (current table is cgroup v1).
- [ ] Amazon Corretto and Azul Zulu images.
- [ ] Non-heap usage of a real Spring Boot app (to back the 350Mi default).
