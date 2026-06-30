default: proto service frontend

proto:
	$(MAKE) -wC proto

service:
	$(MAKE) -wC service

service-unittests:
	$(MAKE) -wC service test

frontend:
	$(MAKE) -wC frontend

integration-tests:
	$(MAKE) -wC integration-tests

docs:
	$(MAKE) -wC docs

buildah:
	buildah bud -t ${REGISTRY_HOSTNAME}:5000/japella-dev
	podman push ${REGISTRY_HOSTNAME}:5000/japella-dev


.PHONY: default proto service service-unittests frontend integration-tests docs
