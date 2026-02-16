# Carbonio base dockerization

This repo contains the compose files of Carbonio.
Every project can include it as a submodule, overriding configs to match their specific needs.

Some services are mocked for the sake of keeping things simple.

## Quick start

Start mails:

```bash
./start-mails.sh
```

Start files:

```bash
./start-files.sh
```

Start tasks:

```bash
./start-tasks.sh
```

Start WSC:

```bash
./start-wsc.sh
```

Start WSC with Videoserver:

```bash
./start-wsc-videoserver.sh
```

Start Admin Panel in isolation:

```bash
./start-admin.sh
```

To start in advanced mode (e.g.: files):

```bash
./start-files.sh --advanced
```
To add monitoring:
```bash
./start-mails.sh --monitoring
```

Stop:

```bash
./stop.sh
```

## How does it work?

The [docker-compose.yaml](docker-compose.yaml) file uses available images for the CE version of
Carbonio.
The [docker-compose-advanced.yaml](docker-compose-advanced.yaml) file contains overrides of the
`docker-compose.yaml`, using advanced images in place of CE images and
adding required settings where needed.

The frontend is assembled in multi-stage Dockerfile (can be found inside
[composed-ui](composed-ui)) which uses Nginx as base image.

Both backend and frontend images can be overridden by using environment
variables (see the compose definition for the respective image).

## Get started

In your project, locate where you want to use the compose of this repository
(let's assume a `docker/` directory).

Include this project as a submodule: `git submodule add https://github.com/zextras/carbonio-base-dockerization.git docker/base/carbonio-base-dockerization`
and initialize it: `git submodule update --init --recursive`.
This will create a carbonio-base-dockerization in your chosen directory.

While inside your chosen directory, you can (and probably want to) create some files:

1. For custom images tags, create a `.env` file, refer to `.env.example` in this project. Default values are "latest", with this you can build using specific images.
2. For custom, maybe locally built images, create a `docker-compose.override.yaml` file. Here you can override existing definitions, refer to `docker-compose.override.example.yaml` in this project.

You can now build everything and start working with
`docker compose --env-file yourpath/.env \
  -f yourpath/carbonio-base-dockerization/docker-compose.yaml \
  -f yourpath/docker-compose.override.yaml \
  up`.
Login using `user@carbonio.localhost`/`assext` or `admin@carbonio.localhost`/`assext` for admin privileges.

You may need to adapt Jenkins to ignore submodules, like this:
`stage('Checkout') {
    steps {
        checkout([
            $class: 'GitSCM',
            branches: scm.branches,
            extensions: [
                [$class: 'SubmoduleOption',
                    disableSubmodules: true,
                    recursiveSubmodules: false
                ]
            ],
            userRemoteConfigs: scm.userRemoteConfigs
        ])
        script {
            env.GIT_COMMIT = sh(script: 'git rev-parse HEAD', returnStdout: true).trim()
        }
    }
}`

## Local DNS Mapping

If `docker.carbonio.localhost` isn’t resolving properly on your machine, try
adding this line to your /etc/hosts file:

```bash
127.0.0.1    docker.carbonio.localhost
```

## Building composed ui

```bash
cd composed-ui
docker build --platform linux/amd64 -t carbonio-composed-ui:local . --no-cache
```

To pull latest UI images be sure to run the same command wih the `--pull`
flag, else the build will use already pulled images and not the latest.

## Mac arm64 Users

To install non-arm64 images, you can use:

```bash
export DOCKER_DEFAULT_PLATFORM=linux/amd64
```

## Building and Running Local Images

If you want to build and run a local Docker image (e.g., for development or testing), follow these steps:

### 1. Build the local image

Navigate to your project directory and build the image with the `linux/amd64` platform (required for Mac arm64 compatibility):

```bash
docker build --platform linux/amd64 -t <image-name>:local .
```

### 2. Run with the local image

Export the image environment variable and run the appropriate start script.

### Example: carbonio-admin-console-ui

```bash
# Build the image
docker build --platform linux/amd64 -t carbonio-admin-console-ui:local .

# Run with the local image
export CARBONIO_ADMIN_CONSOLE_UI_IMAGE=carbonio-admin-console-ui:local
./start-admin.sh --advanced
```

### Available Image Environment Variables

You can override other service images using similar environment variables. Check the compose files for available image variables.
