default: proto service frontend

proto:
	$(MAKE) -wC proto

service:
	$(MAKE) -wC service

frontend:
	$(MAKE) -wC frontend

docs:
	$(MAKE) -wC docs
	./docs/node_modules/.bin/antora antora-playbook.yml


.PHONY: default proto service frontend docs
