default: test-install test-run

test-install:
	npm install --no-fund

test-run:
	npx mocha tests --recursive -t 15000

find-flakey-tests:
	sh -c "while make test-run; do :; done"

clean:
	docker compose down -v
	rm -rf .runtime screenshots

.PHONY: default test-install test-run find-flakey-tests clean
