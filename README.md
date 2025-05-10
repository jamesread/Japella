<div align = "center">
	<img alt = "project logo" src = "logo.png" width = "128" />
	<h1>Japella</h1>
	<p>A social media & chat powertool.</p>

[![Maturity](https://img.shields.io/badge/maturity-Sandbox-yellow)](#none)
[![Build Tag](https://github.com/jamesread/Sicroc/actions/workflows/build-tag.yml/badge.svg)](https://github.com/jamesread/Sicroc/actions/workflows/build-tag.yml)
[![Discord](https://img.shields.io/discord/846737624960860180?label=Discord%20Server)](https://discord.gg/jhYWWpNJ3v)
</div>

Japella's architecture splits out the connections to different protocols (eg Telegram, Discord) into "adaptor" services, and all bot functionality is implemented in a separate service, meaning that the chatbot functionality can easily work with new protocols, by just creating a new adaptor, or, can relay chat messages across protocols, for example.

## Quickstart

Japella is only distributed as a x86_64 Linux container, and requires a RabbitMQ server as the message queue.

There is ready-to-go docker-compose.yml file here: https://jamesread.github.io/Japella/installation/docker-compose/

## Helpful links

* [Japella Documentation](https://jamesread.github.io/Japella/)
