# Japella

An extensible chatbot application, built with containers connected to a message queue.

Japella's architecture splits out the connections to different protocols (eg Telegram, Discord) into "adaptor" services, and all bot functionality is implemented in a separate service, meaning that the chatbot functionality can easily work with new protocols, by just creating a new adaptor, or, can relay chat messages across protocols, for example. 

## Quickstart

Japella is only distributed as a x86_64 Linux container, and requires a RabbitMQ server as the message queue.

````
user@host: docker create --name rabbitmq docker.io/rabbitmq -P 5672:5672
user@host: docker create --name japella-telegram ghcr.io/jamesread/japella-adaptor-telegram
````

## `config.common.yaml`

```yaml
amqp:
	host: localhost
	user: guest
	pass: guest
	exchange: japella
	port: 5672
```
