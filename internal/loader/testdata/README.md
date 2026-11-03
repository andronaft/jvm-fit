Fixtures for rendering tests:

- `spring-app/`: a Helm chart in the style of the Bitnami charts (resource
  presets, `extraEnvVars`, `_helpers.tpl` names). The default values give a
  Java container with a 768Mi limit, 750m CPU and `-Xmx600m`.
- `kustomize/`: a base and a `prod` overlay that patches the memory limit
  and `JAVA_TOOL_OPTIONS`. Only the overlay output is meaningful; the base
  alone has no limits.
