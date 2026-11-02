package flags

import (
	"path"
	"regexp"
	"strings"
)

// Env vars the JVM itself reads, in the order it applies them.
const (
	EnvToolOptions = "JAVA_TOOL_OPTIONS" // read by every JVM (HotSpot and OpenJ9)
	EnvJDKOptions  = "JDK_JAVA_OPTIONS"  // read by the java launcher, JDK 9+
)

// DefaultAssumedEnv are env vars that many images and start scripts pass to
// java, but the JVM does not read on its own.
var DefaultAssumedEnv = []string{"JAVA_OPTS", "JAVA_OPTIONS", "JVM_OPTS", "CATALINA_OPTS"}

// Input is everything in a container spec that can carry JVM options.
type Input struct {
	Env     []EnvVar
	Command []string
	Args    []string
	// AssumedEnv overrides DefaultAssumedEnv when not nil.
	AssumedEnv []string
}

// EnvVar is a container env var with a literal value. Vars set from
// secrets or config maps have Value "" and FromRef true.
type EnvVar struct {
	Name    string
	Value   string
	FromRef bool
}

// Result is the parsed flags plus notes about what could not be read.
type Result struct {
	Flags    Flags
	Warnings []Warning
	// Unresolved lists env vars that may hold JVM options but come from a
	// ConfigMap or Secret, so their value is not in the manifest.
	Unresolved []string
}

var shells = map[string]bool{"sh": true, "bash": true, "ash": true, "dash": true, "zsh": true}

// valueOptions are java launcher options that take the next word as a value.
var valueOptions = map[string]bool{
	"-cp": true, "-classpath": true, "--class-path": true,
	"-p": true, "--module-path": true, "--upgrade-module-path": true,
	"--add-modules": true, "--add-opens": true, "--add-exports": true, "--add-reads": true,
	"--patch-module": true, "--limit-modules": true, "--enable-native-access": true,
}

// Collect gathers JVM options from env vars, command and args in the order the
// JVM applies them: JAVA_TOOL_OPTIONS, JDK_JAVA_OPTIONS, assumed vars such as
// JAVA_OPTS, then the command line.
func Collect(in Input) Result {
	var res Result
	env := map[string]string{}
	for _, e := range in.Env {
		if e.FromRef {
			continue
		}
		env[e.Name] = ExpandKube(e.Value, env)
	}
	assumed := in.AssumedEnv
	if assumed == nil {
		assumed = DefaultAssumedEnv
	}

	cmdline := append(append([]string{}, in.Command...), in.Args...)
	for i := range cmdline {
		cmdline[i] = ExpandKube(cmdline[i], env)
	}
	referenced := func(name string) bool {
		re := regexp.MustCompile(`\$(\{` + name + `\}|` + name + `\b|\(` + name + `\))`)
		for _, s := range append(append([]string{}, in.Command...), in.Args...) {
			if re.MatchString(s) {
				return true
			}
		}
		return false
	}

	apply := func(tokens []string, src Source) {
		res.Warnings = append(res.Warnings, res.Flags.Apply(tokens, src)...)
	}
	for _, e := range in.Env {
		if !e.FromRef {
			continue
		}
		if e.Name == EnvToolOptions || e.Name == EnvJDKOptions || contains(assumed, e.Name) {
			res.Unresolved = append(res.Unresolved, e.Name)
		}
	}
	if v, ok := env[EnvToolOptions]; ok {
		apply(Split(v), Source{Name: EnvToolOptions})
	}
	if v, ok := env[EnvJDKOptions]; ok {
		apply(Split(v), Source{Name: EnvJDKOptions})
	}
	for _, name := range assumed {
		v, ok := env[name]
		if !ok || referenced(name) {
			// A var the command uses directly is applied with the command line.
			continue
		}
		apply(Split(v), Source{Name: name, Assumed: true})
	}

	tokens, src := commandTokens(cmdline, env)
	apply(jvmOptions(tokens), src)
	return res
}

// commandTokens turns command+args into words, unwrapping "sh -c <script>".
func commandTokens(cmdline []string, env map[string]string) ([]string, Source) {
	if len(cmdline) >= 3 && shells[path.Base(cmdline[0])] {
		for i := 1; i < len(cmdline)-1; i++ {
			if strings.HasPrefix(cmdline[i], "-") && strings.Contains(cmdline[i], "c") {
				script := ExpandShell(cmdline[i+1], env)
				return Split(script), Source{Name: "command"}
			}
		}
	}
	return cmdline, Source{Name: "command"}
}

// jvmOptions returns the words that the JVM treats as its own options: the
// ones after "java" and before the main class or jar. Without a "java" word
// (the image entrypoint starts java) every word is a candidate.
func jvmOptions(words []string) []string {
	start := -1
	for i, w := range words {
		if path.Base(w) == "java" {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return words
	}
	var out []string
	for i := start; i < len(words); i++ {
		w := words[i]
		switch {
		case w == "-jar" || w == "-m" || w == "--module" || w == "--source":
			return out
		case valueOptions[w]:
			i++
		case strings.HasPrefix(w, "-"):
			out = append(out, w)
		default:
			// Main class: everything after it belongs to the app.
			return out
		}
		// Shell separators end the java command.
		if w == "&&" || w == ";" || w == "|" {
			return out
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
