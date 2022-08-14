This is a Go solution for the
["Build Your Own Docker" Challenge](https://codecrafters.io/challenges/docker).

This program pulls an image from [Docker Hub](https://hub.docker.com/) and execute commands in it.

# Prerequisites

It uses linux-specific syscalls so we have to run it _inside_ a Docker container.

Please ensure you have [Docker installed](https://docs.docker.com/get-docker/)
locally.

# How to run it

Add a [shell alias](https://shapeshed.com/unix-alias/):

```sh
alias mydocker='docker build -t mydocker . && docker run --cap-add="SYS_ADMIN" mydocker'
```

(The `--cap-add="SYS_ADMIN"` flag is required to create
[PID Namespaces](https://man7.org/linux/man-pages/man7/pid_namespaces.7.html))

Now you can execute program:

```sh
mydocker run alpine:latest echo hey
```

# How to run tests

CGO_ENABLED is disabled because tests use gcc by default which isn't installed in alpine golang image

```sh
docker run --cap-add "SYS_ADMIN" --rm \
           -v `pwd`/app:/app \
           -w '/app' \
           --env "CGO_ENABLED=0" \
           golang:1.19-alpine \
           go test docker/registry
```
