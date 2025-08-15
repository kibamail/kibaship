.PHONY: dev help

dev:
	docker compose up --wait
	pnpm dev

help:
	@echo "Available commands:"
	@echo "  make dev    - Start MySQL service and run development server"
	@echo "  make help   - Show this help message"

# Default target
.DEFAULT_GOAL := help
