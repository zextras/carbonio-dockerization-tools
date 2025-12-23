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

This tool can also work programmatically by using the argument:

```--config-file path/to/config.yaml```

And as a debug feature a command to save the logs locally can also be passed:

```--save-logs```

---

### Assumptions
All the assumptions on the current base dockerization structure are defined in the ```docker_config.go``` file.

1) Some services are considered basic and cannot be disabled (mailbox&deps)
2) Some services are built locally, so the tag cannot be changed
3) Registry is fixed so no personal images can be passed (as for now)
4) Consul registrators are mapped with their service and considered basic for the service, so they cannot be disabled and are not shown in the list
5) Cleanup is always performed on exit/on start

### Project structure
The CLI includes the entirety of carbonio-dockerization inside the embedded directory using
a git subtree and thus the dockerization can be updated with:

```git subtree pull --prefix=internal/embedded/carbonio-dockerization carbonio-dockerization devel --squash```

This is really useful for compiling a self-extracting file that can then work on a static dockerization without it being subject to updates that may break it.

### Releases
Releases are handled by a Github workflow, so every time something is merged on the devel branch a release will be triggered
(using semantic release for version calculation and tag).

Compiled files are then published on github linked to the tag itself and thus can be downloaded from there.