default: proto service frontend

proto:
	$(MAKE) -wC proto

service:
	$(MAKE) -wC service

service-unittests:
	$(MAKE) -wC service test

test: service-unittests

lint:
	$(MAKE) -wC service lint
	$(MAKE) -wC frontend lint

frontend:
	$(MAKE) -wC frontend

it: integration-tests

integration-tests:
	$(MAKE) -wC integration-tests

docs:
	$(MAKE) -wC docs

screenshots:
	$(MAKE) -wC var/marketing/screenshots

buildah:
	buildah bud -t ${REGISTRY_HOSTNAME}:5000/japella-dev
	podman push ${REGISTRY_HOSTNAME}:5000/japella-dev


.PHONY: default proto service service-unittests test lint frontend integration-tests docs screenshots
