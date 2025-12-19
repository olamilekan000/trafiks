# Helm chart deployment Makefile for Trafiks
TRAFIK_RELEASE_NAME = trafiks
NAMESPACE ?= default

# Upgrade Trafiks application
upgrade:
	@echo "Upgrading Trafiks..."
	@helm upgrade $(TRAFIK_RELEASE_NAME) . --namespace $(NAMESPACE)

# Uninstall Trafiks application
uninstall:
	@echo "Uninstalling Trafiks..."
	@helm uninstall $(TRAFIK_RELEASE_NAME) --namespace $(NAMESPACE)

# Install both Traefik and Trafiks
install-all: install-traefik install
	@echo "Traefik and Trafiks installed successfully!"

# Upgrade both Traefik and Trafiks
upgrade-all: upgrade-traefik upgrade
	@echo "Traefik and Trafiks upgraded successfully!"


# Show Trafiks status
status:
	@echo "Trafiks status:"
	@helm status $(TRAFIK_RELEASE_NAME) --namespace $(NAMESPACE) || echo "Trafiks not installed"

# Show status of both
status-all: status-traefik status

# Lint Helm chart
lint:
	@echo "Linting Helm chart..."
	@helm lint .

# Template Helm chart (dry-run)
template:
	@echo "Rendering Helm templates..."
	@helm template $(TRAFIK_RELEASE_NAME) . --namespace $(NAMESPACE)

# Show values
values:
	@helm show values . || cat values.yaml

.PHONY: install upgrade uninstall install-all upgrade-all status status-all lint template values
