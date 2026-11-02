package detect

import (
	"testing"

	"github.com/andronaft/jvm-fit/internal/loader"
)

func TestContainer(t *testing.T) {
	tests := []struct {
		name    string
		c       loader.Container
		java    bool
		runtime Runtime
		major   int
	}{
		{"temurin", loader.Container{Image: "eclipse-temurin:17-jre"}, true, HotSpot, 17},
		{"temurin with registry", loader.Container{Image: "docker.io/library/eclipse-temurin:21.0.2_13-jdk"}, true, HotSpot, 21},
		{"corretto", loader.Container{Image: "amazoncorretto:11"}, true, HotSpot, 11},
		{"ecr corretto", loader.Container{Image: "public.ecr.aws/amazoncorretto/amazoncorretto:21-al2023"}, true, HotSpot, 21},
		{"semeru", loader.Container{Image: "ibm-semeru-runtimes:open-17-jre"}, true, OpenJ9, 17},
		{"zulu", loader.Container{Image: "azul/zulu-openjdk:17"}, true, HotSpot, 17},
		{"distroless", loader.Container{Image: "gcr.io/distroless/java17-debian12"}, true, HotSpot, 0},
		{"tomcat", loader.Container{Image: "tomcat:10.1-jdk17"}, true, HotSpot, 17},
		{"openjdk 8", loader.Container{Image: "openjdk:8u342-jre"}, true, HotSpot, 8},
		{"app image with java command", loader.Container{Image: "ghcr.io/acme/payments:1.4.2", Command: []string{"java", "-jar", "/app.jar"}}, true, HotSpot, 0},
		{"app image with sh -c java", loader.Container{Image: "acme/api:3", Command: []string{"sh", "-c", "exec java $JAVA_OPTS -jar app.jar"}}, true, HotSpot, 0},
		{"app image with JAVA_TOOL_OPTIONS", loader.Container{Image: "acme/api:3", Env: []loader.EnvVar{{Name: "JAVA_TOOL_OPTIONS", Value: "-Xmx1g"}}}, true, HotSpot, 0},
		{"openj9 by flag", loader.Container{Image: "acme/api:3", Env: []loader.EnvVar{{Name: "JAVA_OPTS", Value: "-Xgcpolicy:gencon"}}}, true, OpenJ9, 0},
		{"nginx", loader.Container{Image: "nginx:1.27"}, false, HotSpot, 0},
		{"javascript is not java", loader.Container{Image: "node:20", Command: []string{"node", "server.js"}}, false, HotSpot, 0},
		{"redis", loader.Container{Image: "redis:7"}, false, HotSpot, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Container(tt.c)
			if r.Java != tt.java {
				t.Fatalf("Java = %v, want %v (reasons %v)", r.Java, tt.java, r.Reasons)
			}
			if !tt.java {
				return
			}
			if r.Runtime != tt.runtime {
				t.Errorf("Runtime = %s, want %s", r.Runtime, tt.runtime)
			}
			if r.Major != tt.major {
				t.Errorf("Major = %d, want %d", r.Major, tt.major)
			}
		})
	}
}
