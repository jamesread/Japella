# Docker

Japella is best run in a Docker container.

!!! warning
    **The recommended way to run Japella is** [docker compose](docker-compose.md).

	Using Japella with a simple `docker create` command is possible, but it makes upgrades and scaling a bit of a pain.


```bash
$ docker create --name japella -v japella:/config ghcr.io/jamesread/japella:latest
$ docker start japella
```
