GORELEASER_VERSION ?= v2.11.2
GORELEASER = go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
GOFLAGS = -buildvcs=false

.PHONY: build format-check vet test release-check snapshot install-smoke setup verify verify-skills index install-hooks

build:
	go build $(GOFLAGS) ./...

format-check:
	test -z "$$(gofmt -l .)"

vet:
	go vet ./...

test:
	go test $(GOFLAGS) ./...

release-check:
	$(GORELEASER) check

snapshot:
	$(GORELEASER) release --snapshot --clean

install-smoke: snapshot
	./scripts/install-smoke.sh

setup:
	go run $(GOFLAGS) ./cmd/jstack setup

verify-skills:
	python3 scripts/verify.py

verify: format-check build vet test verify-skills

index:
	python3 scripts/build-index.py

install-hooks:
	@command -v gitleaks >/dev/null || { echo "gitleaks is required: https://github.com/gitleaks/gitleaks"; exit 1; }
	git config core.hooksPath .githooks
	@echo "Installed the pre-commit hook: gitleaks, then scripts/verify.py."
