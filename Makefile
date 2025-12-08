APP_NAME    := pda-monitor
BIN_DIR     := ./bin

GO          := go

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

# Dev-only run (loads .env if present)
run: build
	@if [ -f .env ]; then \
		export $$(cat .env | grep -v '^#' | xargs) && \
		$(BIN_DIR)/$(APP_NAME); \
	else \
		echo "WARNING: .env not found, running without env"; \
		$(BIN_DIR)/$(APP_NAME); \
	fi

install-user: build
	mkdir -p $$HOME/.local/bin
	mkdir -p $$HOME/.local/share/pda-monitor
	mkdir -p $$HOME/.config/pda-monitor
	cp $(BIN_DIR)/$(APP_NAME) $$HOME/.local/bin/$(APP_NAME)
	chmod +x $$HOME/.local/bin/$(APP_NAME)
	@echo "Binary installed to $$HOME/.local/bin/$(APP_NAME)"
	@echo "Put your runtime config in $$HOME/.config/pda-monitor/.env"

clean:
	rm -rf $(BIN_DIR)