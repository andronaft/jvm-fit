#!/usr/bin/env bash
# Run real JVMs inside Docker with different memory and CPU limits and print
# what the JVM picks: max heap, active processor count and garbage collector.
#
# Usage: docs/evidence/run.sh > docs/evidence/results.md
#
# Needs Docker. The first run pulls the images (a few hundred MB each).
set -euo pipefail

IMAGES=(
  "eclipse-temurin:11-jre"
  "eclipse-temurin:17-jre"
  "eclipse-temurin:21-jre"
  "ibm-semeru-runtimes:open-21-jre"
)

# memory|cpus  ("none" = no limit)
COMBOS=(
  "256m|0.5"
  "512m|0.5"
  "512m|1"
  "1g|1"
  "1g|2"
  "2g|1"
  "2g|2"
  "4g|4"
  "none|none"
)

# Extra flag sets to show how flags change the result.
# They run only for the combos in FLAG_COMBOS, to keep the table short.
FLAGSETS=(
  ""
  "-XX:MaxRAMPercentage=75.0"
  "-XX:ActiveProcessorCount=2"
  "-XX:-UseContainerSupport"
)
FLAG_COMBOS=("1g|1" "2g|1")

docker_args() {
  local mem="$1" cpus="$2" args=()
  [[ "$mem" != "none" ]] && args+=("--memory=$mem")
  [[ "$cpus" != "none" ]] && args+=("--cpus=$cpus")
  echo "${args[@]:-}"
}

# Prints "heap_MiB|cpus|gc" for one JVM run.
hotspot_probe() {
  local image="$1" mem="$2" cpus="$3" flags="$4"
  # shellcheck disable=SC2046
  docker run --rm $(docker_args "$mem" "$cpus") "$image" sh -c "
    out=\$(java $flags -XX:+PrintFlagsFinal -version 2>/dev/null)
    heap=\$(echo \"\$out\" | awk '\$2==\"MaxHeapSize\"{printf \"%d\", \$4/1048576}')
    apc=\$(echo \"\$out\" | awk '\$2==\"ActiveProcessorCount\"{print \$4}')
    gc=\$(echo \"\$out\" | awk '\$4==\"true\" && \$2 ~ /^Use(Serial|Parallel|G1|Z|Shenandoah)GC\$/ {print \$2}' | paste -sd, -)
    cpu=\$(java $flags -XshowSettings:system -version 2>&1 | awk -F': ' '/Effective CPU Count/{print \$2; exit}')
    [ \"\$apc\" != \"-1\" ] && cpu=\"\$apc (flag)\"
    echo \"\$heap|\${cpu:-?}|\$gc\""
}

# OpenJ9 has no PrintFlagsFinal; -verbose:sizes prints the -Xmx it picked.
openj9_probe() {
  local image="$1" mem="$2" cpus="$3" flags="$4"
  # shellcheck disable=SC2046
  docker run --rm $(docker_args "$mem" "$cpus") "$image" sh -c "
    heap=\$(java $flags -verbose:sizes -version 2>&1 | awk '\$1 ~ /^-Xmx/ {sub(/^-Xmx/, \"\", \$1); print \$1; exit}')
    cpu=\$(java $flags -XshowSettings:system -version 2>&1 | awk -F': ' '/Effective CPU Count/{print \$2; exit}')
    echo \"\$heap|\${cpu:-?}|gencon (default)\""
}

echo "Docker $(docker version --format '{{.Server.Version}}'), cgroup v$(docker info --format '{{.CgroupVersion}}'), host CPUs $(docker info --format '{{.NCPU}}'), host memory $(( $(docker info --format '{{.MemTotal}}') / 1048576 ))MiB, arch $(docker info --format '{{.Architecture}}')"
echo
echo "Date: $(date -u +%Y-%m-%d)"
echo
echo "| image | --memory | --cpus | flags | max heap | JVM CPUs | GC |"
echo "|---|---|---|---|---|---|---|"
for image in "${IMAGES[@]}"; do
  docker pull -q "$image" >/dev/null
  version=$(docker run --rm "$image" java -version 2>&1 | awk -F'"' '/version/{print $2; exit}')
  for combo in "${COMBOS[@]}"; do
    mem="${combo%%|*}"; cpus="${combo##*|}"
    for flags in "${FLAGSETS[@]}"; do
      if [[ -n "$flags" && ! " ${FLAG_COMBOS[*]} " =~ " $combo " ]]; then continue; fi
      if [[ "$image" == *semeru* ]]; then
        row=$(openj9_probe "$image" "$mem" "$cpus" "$flags")
        IFS='|' read -r heap cpu gc <<<"$row"
      else
        row=$(hotspot_probe "$image" "$mem" "$cpus" "$flags")
        IFS='|' read -r heap cpu gc <<<"$row"
        heap="${heap}M"
      fi
      echo "| $image ($version) | $mem | $cpus | ${flags:--} | $heap | $cpu | $gc |"
    done
  done
done
