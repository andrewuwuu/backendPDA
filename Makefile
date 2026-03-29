APP_NAME    := pda-monitor
DBINIT_NAME := pda-dbinit
BIN_DIR     := ./bin

GO          := go

build: build-server

build-server:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

build-dbinit:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(DBINIT_NAME) ./cmd/dbinit

# Dev-only run (loads .env if present)
run: build-server
	@if [ -f .env ]; then \
		set -a; \
		. ./.env; \
		set +a; \
		$(BIN_DIR)/$(APP_NAME); \
	else \
		echo "WARNING: .env not found, running without env"; \
		$(BIN_DIR)/$(APP_NAME); \
	fi

install-user: build-server build-dbinit
	mkdir -p $$HOME/.local/bin
	mkdir -p $$HOME/.local/share/pda-monitor
	mkdir -p $$HOME/.config/pda-monitor
	cp $(BIN_DIR)/$(APP_NAME) $$HOME/.local/bin/$(APP_NAME)
	cp $(BIN_DIR)/$(DBINIT_NAME) $$HOME/.local/bin/$(DBINIT_NAME)
	chmod +x $$HOME/.local/bin/$(APP_NAME)
	chmod +x $$HOME/.local/bin/$(DBINIT_NAME)
	@echo "Binary installed to $$HOME/.local/bin/$(APP_NAME)"
	@echo "Binary installed to $$HOME/.local/bin/$(DBINIT_NAME)"
	@echo "Put your runtime config in $$HOME/.config/pda-monitor/.env"

clean:
	rm -rf $(BIN_DIR)
