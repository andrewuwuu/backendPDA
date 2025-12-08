APP_NAME    := pda-monitor
BIN_DIR     := ./bin

GO          := go

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

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