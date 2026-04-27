# External Plugin Example

Minimal external plugin that adds `terraci hello` command. Demonstrates the plugin registration, config, and command patterns needed to extend TerraCi.

## What it does

- Registers a `hello` plugin with optional `greeting` config
- Adds `terraci hello` command that scans and lists Terraform modules
- Shows how to use `AppContext`, `discovery.Scanner`, and `config.ParsePattern`

## Build

```bash
# From repository root
xterraci build \
  --with github.com/edelwud/terraci/examples/external-plugin=./examples/external-plugin \
  --output ./build/terraci-hello

# Verify
./build/terraci-hello hello
```

## Optional config

```yaml
# .terraci.yaml
plugins:
  hello:
    greeting: "Hi from my custom plugin!"
```

## Plugin structure

```
external-plugin/
├── plugin.go      # Plugin struct, Config, init() registration
├── commands.go    # CommandProvider — terraci hello
├── go.mod         # Separate Go module
└── README.md
```

## Key patterns

1. **Registration**: `registry.RegisterFactory()` in `init()` — blank import triggers factory registration
2. **BasePlugin[C]**: Generic embedding gives each command-run plugin instance config loading and enable/disable behavior
3. **CommandProvider**: Return `[]*cobra.Command` from `Commands(ctx)` — framework adds them to CLI
4. **AppContext**: Access config, working directory, service directory at command time
