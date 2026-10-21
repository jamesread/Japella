# Regenerates marketing screenshots against a throwaway stack seeded with
# fictional data. Run `make` here; everything is torn down afterwards.

ROOT := $(abspath ../../..)
HERE := $(abspath .)
TMP := $(HERE)/tmp
RUN := $(TMP)/run

CONTAINER ?= podman
DB_CTR := japella-mkt-mariadb
DB_PORT := 13307

BACKEND_URL := http://127.0.0.1:18380

DOCS_IMAGES := $(ROOT)/docs/modules/ROOT/images/marketing

# Japella reads ~/.config/japella/config.yaml, so the backend runs with HOME
# pointed into tmp/ and never sees a developer's own config. Outbound HTTP goes
# to a closed local port, so the (token-less) demo accounts can never reach a
# real social network, even if a background job fires mid-run.
BACKEND_ENV := HOME=$(TMP)/home TZ=UTC \
	HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 NO_PROXY=127.0.0.1,localhost \
	JAPELLA_SECURE_COOKIES=false \
	JAPELLA_LOG_FILE=$(TMP)/japella-app.log

.PHONY: all build up seed shoot manifest publish down clean

all:
	@$(MAKE) build up seed shoot manifest publish; rc=$$?; $(MAKE) down; exit $$rc

build:
	$(MAKE) -C $(ROOT)/service
	[ -d $(ROOT)/frontend/node_modules ] || $(MAKE) -C $(ROOT)/frontend deps
	$(MAKE) -C $(ROOT)/frontend prod

up:
	mkdir -p $(TMP)/home/.config/japella $(RUN)/service/media $(RUN)/var
	cp service.yaml $(TMP)/home/.config/japella/config.yaml
	# The backend finds migrations, languages and the SPA relative to its cwd.
	ln -sfn $(ROOT)/var/app-skel $(RUN)/var/app-skel
	ln -sfn $(ROOT)/frontend $(RUN)/frontend
	$(CONTAINER) run -d --rm --name $(DB_CTR) -p 127.0.0.1:$(DB_PORT):3306 \
		-e MARIADB_ROOT_PASSWORD=root-demo -e MARIADB_DATABASE=japella \
		-e MARIADB_USER=japella -e MARIADB_PASSWORD=japella-demo \
		docker.io/library/mariadb:11.4
	@echo "Waiting for MariaDB..."
	@for i in $$(seq 120); do \
		$(CONTAINER) exec $(DB_CTR) mariadb -ujapella -pjapella-demo -h127.0.0.1 -e 'SELECT 1' japella >/dev/null 2>&1 && exit 0; \
		sleep 1; done; echo "MariaDB did not start"; exit 1
	cd $(RUN)/service && { $(BACKEND_ENV) $(ROOT)/service/japella > $(TMP)/service.log 2>&1 & echo $$! > $(TMP)/service.pid; }
	# /healthz passes before the DB is migrated; API calls only succeed once
	# the database and connector controller are ready.
	@for i in $$(seq 90); do curl -fsS -X POST -H 'Content-Type: application/json' -d '{}' \
		$(BACKEND_URL)/api/japella.controlapi.v1.JapellaControlApiService/GetCvars >/dev/null 2>&1 && exit 0; \
		curl -s -o /dev/null -w '%{http_code}' -X POST -H 'Content-Type: application/json' -d '{}' \
		$(BACKEND_URL)/api/japella.controlapi.v1.JapellaControlApiService/GetCvars | grep -q '^401$$' && exit 0; \
		sleep 1; done; echo "Backend did not become ready; see $(TMP)/service.log"; exit 1

seed:
	SQL="$(CONTAINER) exec -i $(DB_CTR) mariadb -ujapella -pjapella-demo japella" \
		./seed.sh $(BACKEND_URL) $(TMP)

# The backend and DB run in UTC; so does the browser, so times look the same
# on every machine.
shoot:
	TZ=UTC repo-helper screenshot --config screenshots.ini

manifest:
	./generate-manifest.py

publish:
	mkdir -p $(DOCS_IMAGES)
	cp *.png $(DOCS_IMAGES)/

down:
	-@[ -f $(TMP)/service.pid ] && kill $$(cat $(TMP)/service.pid) 2>/dev/null; rm -f $(TMP)/service.pid
	-@$(CONTAINER) rm -f $(DB_CTR) >/dev/null 2>&1

clean: down
	rm -rf $(TMP)
