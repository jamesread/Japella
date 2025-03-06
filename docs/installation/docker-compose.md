# Docker Compose

Docker compose is the recommended way to run Japella. It allows you to easily maintain your japella installation consistently across upgrades.

You can use the following `docker-compose.yml` file to run Japella.

```yaml title="docker-compose.yml"
---
services:
  japella:
    container_name: japella
    image: ghcr.io/jamesread/japella
    volumes:
      - japella-config:/config
    restart: unless-stopped
```
