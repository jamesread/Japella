default: proto service frontend

proto:
	$(MAKE) -wC proto

service:
	$(MAKE) -wC service

frontend:
	$(MAKE) -wC frontend

docs:
	mkdocs serve


.PHONY: default proto service frontend docs
