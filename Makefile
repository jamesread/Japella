default: proto service frontend

proto:
	$(MAKE) -wC proto

service:
	$(MAKE) -wC service

frontend:
	$(MAKE) -wC frontend

docs:
	$(MAKE) -wC docs

buildah:
	buildah bud -t ${REGISTRY_HOSTNAME}:5000/japella-dev
	podman push ${REGISTRY_HOSTNAME}:5000/japella-dev


.PHONY: default proto service frontend docs
