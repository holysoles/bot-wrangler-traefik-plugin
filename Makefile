.PHONY: lint test race bench vendor clean

export GO111MODULE=on

default: lint test race

ci: lint test

lint:
	golangci-lint run

test:
	go test -v -cover -coverprofile=coverage.out ./...

race:
	go test -race ./...

bench:
	go test -bench=. ./...

test_codecov:
	cat codecov.yml | curl --data-binary @- https://codecov.io/validate

yaegi_test:
	yaegi test -v .; \
	root=$$(pwd); \
	dirs=$$(find ./pkg -name "*.go" -printf "%h\n" | uniq);\
	for dir in $$dirs; do \
		echo "testing $$dir"; \
		cd "$$dir"; \
		yaegi test -v .; \
        	if [ $$? != 0 ] ; then \
			echo "Yaegi test failed!"; \
			exit 1; \
		fi; \
		cd "$$root"; \
	done

vendor:
	go mod vendor

clean:
	rm -rf ./vendor

run_local:
	docker compose -f docker-compose.local.yml up -d --remove-orphans

restart_local:
	docker compose -f docker-compose.local.yml restart

stop_local:
	docker compose -f docker-compose.local.yml down
