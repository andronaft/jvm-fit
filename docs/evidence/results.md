Docker 20.10.8, cgroup v1, host CPUs 4, host memory 1988MiB, arch aarch64

Date: 2026-11-02

| image | --memory | --cpus | flags | max heap | JVM CPUs | GC |
|---|---|---|---|---|---|---|
| eclipse-temurin:11-jre (11.0.32.1) | 256m | 0.5 | - | 126M | 1 | UseSerialGC |
| eclipse-temurin:11-jre (11.0.32.1) | 512m | 0.5 | - | 128M | 1 | UseSerialGC |
| eclipse-temurin:11-jre (11.0.32.1) | 512m | 1 | - | 128M | 1 | UseSerialGC |
| eclipse-temurin:11-jre (11.0.32.1) | 1g | 1 | - | 256M | 1 | UseSerialGC |
| eclipse-temurin:11-jre (11.0.32.1) | 1g | 1 | -XX:MaxRAMPercentage=75.0 | 768M | 1 | UseSerialGC |
| eclipse-temurin:11-jre (11.0.32.1) | 1g | 1 | -XX:ActiveProcessorCount=2 | 256M | 2 (flag) | UseSerialGC |
| eclipse-temurin:11-jre (11.0.32.1) | 1g | 1 | -XX:-UseContainerSupport | 498M | ? | UseG1GC |
| eclipse-temurin:11-jre (11.0.32.1) | 1g | 2 | - | 256M | 2 | UseSerialGC |
| eclipse-temurin:11-jre (11.0.32.1) | 2g | 1 | - | 498M | 1 | UseSerialGC |
| eclipse-temurin:11-jre (11.0.32.1) | 2g | 1 | -XX:MaxRAMPercentage=75.0 | 1492M | 1 | UseSerialGC |
| eclipse-temurin:11-jre (11.0.32.1) | 2g | 1 | -XX:ActiveProcessorCount=2 | 498M | 2 (flag) | UseG1GC |
| eclipse-temurin:11-jre (11.0.32.1) | 2g | 1 | -XX:-UseContainerSupport | 498M | ? | UseG1GC |
| eclipse-temurin:11-jre (11.0.32.1) | 2g | 2 | - | 498M | 2 | UseG1GC |
| eclipse-temurin:11-jre (11.0.32.1) | 4g | 4 | - | 498M | 4 | UseG1GC |
| eclipse-temurin:11-jre (11.0.32.1) | none | none | - | 498M | 4 | UseG1GC |
| eclipse-temurin:17-jre (17.0.20.1) | 256m | 0.5 | - | 126M | 1 | UseSerialGC |
| eclipse-temurin:17-jre (17.0.20.1) | 512m | 0.5 | - | 128M | 1 | UseSerialGC |
| eclipse-temurin:17-jre (17.0.20.1) | 512m | 1 | - | 128M | 1 | UseSerialGC |
| eclipse-temurin:17-jre (17.0.20.1) | 1g | 1 | - | 256M | 1 | UseSerialGC |
| eclipse-temurin:17-jre (17.0.20.1) | 1g | 1 | -XX:MaxRAMPercentage=75.0 | 768M | 1 | UseSerialGC |
| eclipse-temurin:17-jre (17.0.20.1) | 1g | 1 | -XX:ActiveProcessorCount=2 | 256M | 2 (flag) | UseSerialGC |
| eclipse-temurin:17-jre (17.0.20.1) | 1g | 1 | -XX:-UseContainerSupport | 498M | ? | UseG1GC |
| eclipse-temurin:17-jre (17.0.20.1) | 1g | 2 | - | 256M | 2 | UseSerialGC |
| eclipse-temurin:17-jre (17.0.20.1) | 2g | 1 | - | 498M | 1 | UseSerialGC |
| eclipse-temurin:17-jre (17.0.20.1) | 2g | 1 | -XX:MaxRAMPercentage=75.0 | 1492M | 1 | UseSerialGC |
| eclipse-temurin:17-jre (17.0.20.1) | 2g | 1 | -XX:ActiveProcessorCount=2 | 498M | 2 (flag) | UseG1GC |
| eclipse-temurin:17-jre (17.0.20.1) | 2g | 1 | -XX:-UseContainerSupport | 498M | ? | UseG1GC |
| eclipse-temurin:17-jre (17.0.20.1) | 2g | 2 | - | 498M | 2 | UseG1GC |
| eclipse-temurin:17-jre (17.0.20.1) | 4g | 4 | - | 498M | 4 | UseG1GC |
| eclipse-temurin:17-jre (17.0.20.1) | none | none | - | 498M | 4 | UseG1GC |
| eclipse-temurin:21-jre (21.0.13) | 256m | 0.5 | - | 126M | 1 | UseSerialGC |
| eclipse-temurin:21-jre (21.0.13) | 512m | 0.5 | - | 128M | 1 | UseSerialGC |
| eclipse-temurin:21-jre (21.0.13) | 512m | 1 | - | 128M | 1 | UseSerialGC |
| eclipse-temurin:21-jre (21.0.13) | 1g | 1 | - | 256M | 1 | UseSerialGC |
| eclipse-temurin:21-jre (21.0.13) | 1g | 1 | -XX:MaxRAMPercentage=75.0 | 768M | 1 | UseSerialGC |
| eclipse-temurin:21-jre (21.0.13) | 1g | 1 | -XX:ActiveProcessorCount=2 | 256M | 2 (flag) | UseSerialGC |
| eclipse-temurin:21-jre (21.0.13) | 1g | 1 | -XX:-UseContainerSupport | 498M | ? | UseG1GC |
| eclipse-temurin:21-jre (21.0.13) | 1g | 2 | - | 256M | 2 | UseSerialGC |
| eclipse-temurin:21-jre (21.0.13) | 2g | 1 | - | 498M | 1 | UseSerialGC |
| eclipse-temurin:21-jre (21.0.13) | 2g | 1 | -XX:MaxRAMPercentage=75.0 | 1492M | 1 | UseSerialGC |
| eclipse-temurin:21-jre (21.0.13) | 2g | 1 | -XX:ActiveProcessorCount=2 | 498M | 2 (flag) | UseG1GC |
| eclipse-temurin:21-jre (21.0.13) | 2g | 1 | -XX:-UseContainerSupport | 498M | ? | UseG1GC |
| eclipse-temurin:21-jre (21.0.13) | 2g | 2 | - | 498M | 2 | UseG1GC |
| eclipse-temurin:21-jre (21.0.13) | 4g | 4 | - | 498M | 4 | UseG1GC |
| eclipse-temurin:21-jre (21.0.13) | none | none | - | 498M | 4 | UseG1GC |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 256m | 0.5 | - | 128M | 1 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 512m | 0.5 | - | 256M | 1 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 512m | 1 | - | 256M | 1 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 1g | 1 | - | 512M | 1 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 1g | 1 | -XX:MaxRAMPercentage=75.0 | 768M | 1 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 1g | 1 | -XX:ActiveProcessorCount=2 | 512M | 2 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 1g | 1 | -XX:-UseContainerSupport | 512M | ? | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 1g | 2 | - | 512M | 2 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 2g | 1 | - | 512M | 1 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 2g | 1 | -XX:MaxRAMPercentage=75.0 | 1527360K | 1 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 2g | 1 | -XX:ActiveProcessorCount=2 | 512M | 2 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 2g | 1 | -XX:-UseContainerSupport | 512M | ? | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 2g | 2 | - | 512M | 2 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | 4g | 4 | - | 512M | 4 | gencon (default) |
| ibm-semeru-runtimes:open-21-jre (21.0.12.1) | none | none | - | 512M | 4 | gencon (default) |
