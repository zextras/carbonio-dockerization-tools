# Carbonio Docker CLI

A multiplatform CLI based on [carbonio-dockerization](https://github.com/zextras/carbonio-dockerization) created to simplify a granular startup
process without having to handle 5-line long docker commands.

Download from [here](https://github.com/galvagnimatteo/carbonio-docker-cli/releases).

## What it does

The CLI aims to simplify the startup process of a granular dockerization both for developers and non-devs.

This is achieved by providing two ways of using the CLI:
1) Defining a custom infrastructure (selecting single services at specific tags/versions)
2) Importing a pre-made configuration file

Using the custom mode, a dev can specify exactly what services/UI modules to start
and at what specific tag (the list of these services is obtained by parsing the docker compose files).

Of course, he can then export this specific configuration in a file to be used in the other mode.

A non-dev can then start the CLI and simply import the previously generated config file without having to worry about anything else.

## Arguments

| Argument                            | Description                                                              |
|-------------------------------------|--------------------------------------------------------------------------|
| `--config-file path/to/config.yaml` | Load a configuration file and skip the interactive setup                 |
| `--headless`                        | Run without TUI (requires `--config-file`). Useful for CI/CD or scripts  |
| `--with-clean-database`             | Remove all database volumes before starting (fresh installation)         |
| `--save-logs`                       | Save debug logs to `carbonio-docker-cli.log` in the current directory    |

### Examples

**Interactive mode (TUI):**
```bash
./carbonio-docker-cli
```

**Load a config file with TUI monitoring:**
```bash
./carbonio-docker-cli --config-file my-config.yaml
```

**Headless mode (no TUI, for scripts/CI):**
```bash
./carbonio-docker-cli --config-file my-config.yaml --headless
```

**Fresh installation (clean database):**
```bash
./carbonio-docker-cli --config-file my-config.yaml --with-clean-database
```

**Debug mode:**
```bash
./carbonio-docker-cli --save-logs
```

---

## Features

- **Custom images**: You can specify any Docker image (not just from Zextras registry), including locally built images
- **Disable frontend modules**: Individual UI modules can be disabled during setup
- **Export/Import configurations**: Save your setup to a file and share it with others
- **Clean database option**: Start with a fresh database by removing all persistent data
- **Graceful shutdown**: Press `q` or `Ctrl+C` to stop all containers and cleanup

---

## Working Directory

The CLI extracts the embedded dockerization files to a **system cache directory**:

| OS      | Path                                            |
|---------|-------------------------------------------------|
| Linux   | `~/.cache/carbonio-docker-cli/workdir`          |
| macOS   | `~/Library/Caches/carbonio-docker-cli/workdir`  |
| Windows | `%LocalAppData%\carbonio-docker-cli\workdir`    |

This directory is automatically recreated on each run to ensure a clean state.
