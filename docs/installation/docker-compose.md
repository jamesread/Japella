# Docker Compose

Docker compose is the recommended way to run Japella. It allows you to easily maintain your japella installation consistently across upgrades.

You can use the following `docker-compose.yml` file to run Japella.

## Create the docker-compose.yml file

Create this file in a location that is easy to remember.

```yaml title="docker-compose.yml"
---
services:
  japella:
    container_name: japella
    image: ghcr.io/jamesread/japella
    volumes:
      - japella-config:/config
    restart: unless-stopped

  rabbitmq:
    container_name: rabbitmq
    image: docker.io/rabbitmq
    hostname: rabbitmq

# The database in Japella 2 is entirely optional.
#  mariadb:
#    container_name: mariadb
#    image: docker.io/mariadb
#    environment:
#      MARIADB_ROOT_PASSWORD: password
#      MARIADB_DATABASE: japella

volumes:
  japella-config:
    name: japella-config
    external: false
```

## docker compose up

Open a terminal in the same directory as your new `docker-compose.yml` file and run the following command;

```bash
$ docker compose up
```
