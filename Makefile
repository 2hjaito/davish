.PHONY: dev build check lint test test-integration frontend-install sshd-up sshd-down sshd-keys

FRONTEND := frontend
COMPOSE  := docker compose -f test/sshd/compose.yml

dev:
	wails dev

build:
	wails build

frontend-install:
	cd $(FRONTEND) && npm ci

lint:
	go vet ./...
	golangci-lint run
	cd $(FRONTEND) && npm run check && npm run lint

test:
	go test -race ./...
	cd $(FRONTEND) && npm run test

check: lint test

test-integration:
	go test -race -tags=integration ./...

# Test keys are not committed; generate a throwaway pair on first use.
sshd-keys:
	@test -f test/sshd/keys/client_ed25519 || { \
		mkdir -p test/sshd/keys && \
		ssh-keygen -q -t ed25519 -N '' -C davish-test-client -f test/sshd/keys/client_ed25519 && \
		ssh-keygen -q -t ed25519 -N '' -C davish-test-host -f test/sshd/keys/host_ed25519; }

sshd-up: sshd-keys
	$(COMPOSE) up -d --build --wait

sshd-down:
	$(COMPOSE) down
