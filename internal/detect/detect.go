// Package detect decides whether a container runs a JVM and which one.
package detect

import (
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/andronaft/jvm-fit/internal/flags"
	"github.com/andronaft/jvm-fit/internal/loader"
)

// Runtime is the JVM implementation.
type Runtime string

const (
	HotSpot Runtime = "hotspot"
	OpenJ9  Runtime = "openj9"
)

// Result says why a container looks like Java and what is known about the JVM.
type Result struct {
	Java    bool
	Reasons []string
	Runtime Runtime
	// Major is the Java version from the image tag (11, 17, 21), or 0 if unknown.
	Major int
	// Musl: the image name or tag says it is Alpine or musl based, so malloc
	// is musl, not glibc. false also means unknown.
	Musl bool
}

// javaImages match image names (without registry and tag) of JDK/JRE images
// and of common Java app servers.
var javaImages = []*regexp.Regexp{
	regexp.MustCompile(`(^|/)(eclipse-)?temurin$`),
	regexp.MustCompile(`(^|/)(amazon)?corretto$`),
	regexp.MustCompile(`(^|/)openjdk$`),
	regexp.MustCompile(`(^|/)ibm-semeru-runtimes$`),
	regexp.MustCompile(`(^|/)(adoptopenjdk|semeru|open-liberty|websphere-liberty)(/|$)`),
	regexp.MustCompile(`(^|/)azul/zulu-openjdk`),
	regexp.MustCompile(`(^|/)bellsoft/liberica`),
	regexp.MustCompile(`(^|/)graalvm/`),
	regexp.MustCompile(`(^|/)sapmachine$`),
	regexp.MustCompile(`(^|/)microsoft/openjdk`),
	regexp.MustCompile(`(^|/)distroless/java`),
	regexp.MustCompile(`(^|/)ubi\d*/openjdk`),
	regexp.MustCompile(`(^|/)(tomcat|jetty|wildfly|keycloak|kafka|zookeeper|elasticsearch|logstash|jenkins|sonarqube|nexus3|cassandra|solr)$`),
	regexp.MustCompile(`(^|[/-])(jdk|jre|java)([/-]|$)`),
}

// javaEnv are env vars that mark a Java container, besides the ones the JVM
// reads itself (flags.IsJVMEnv).
var javaEnv = map[string]bool{
	"JAVA_OPTS":    true,
	"JAVA_OPTIONS": true,
	"JVM_OPTS":     true,
	"JAVA_HOME":    true,
}

var openj9Image = regexp.MustCompile(`semeru|openj9|liberty|websphere`)

// muslImage matches images built on Alpine or another musl libc:
// eclipse-temurin:21-jre-alpine, bellsoft/liberica-openjdk-alpine-musl.
var muslImage = regexp.MustCompile(`(^|[^a-z])(alpine|musl)([^a-z]|$)`)

// tagVersion finds the Java major version in a tag: 17-jre, 21.0.2_13-jdk, 11, jdk17, java21, 8u402.
var tagVersion = regexp.MustCompile(`(?:^|[^0-9])(?:jdk|jre|java|open-)?(8|1[1-9]|2[0-9])(?:$|[^0-9])`)

// Container checks one container.
func Container(c loader.Container) Result {
	var r Result
	name, tag := splitImage(c.Image)
	for _, re := range javaImages {
		if re.MatchString(name) {
			r.Reasons = append(r.Reasons, "image "+c.Image)
			break
		}
	}
	for _, w := range append(append([]string{}, c.Command...), c.Args...) {
		if path.Base(firstWord(w)) == "java" || strings.Contains(w, "java ") || strings.HasSuffix(w, ".jar") {
			r.Reasons = append(r.Reasons, "command runs java")
			break
		}
	}
	for _, e := range c.Env {
		if javaEnv[e.Name] || flags.IsJVMEnv(e.Name) {
			r.Reasons = append(r.Reasons, "env "+e.Name)
			break
		}
	}
	r.Java = len(r.Reasons) > 0
	r.Runtime = HotSpot
	if openj9Image.MatchString(c.Image) {
		r.Runtime = OpenJ9
	}
	for _, e := range c.Env {
		if flags.IsOpenJ9Env(e.Name) || strings.Contains(e.Value, "-Xgcpolicy:") || strings.Contains(e.Value, "-Xshareclasses") {
			r.Runtime = OpenJ9
		}
	}
	r.Musl = muslImage.MatchString(name) || muslImage.MatchString(tag)
	if m := tagVersion.FindStringSubmatch(tag); m != nil {
		r.Major, _ = strconv.Atoi(m[1])
	}
	return r
}

// splitImage returns the repository path without registry, and the tag.
func splitImage(image string) (name, tag string) {
	name = image
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}
	if i := strings.LastIndex(name, ":"); i > strings.LastIndex(name, "/") {
		name, tag = name[:i], name[i+1:]
	}
	if i := strings.Index(name, "/"); i >= 0 && strings.ContainsAny(name[:i], ".:") {
		name = name[i+1:]
	}
	return strings.ToLower(name), strings.ToLower(tag)
}

func firstWord(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}
