# Helm chart deployment Makefile for Trafiks

HELM_REPO_TRAEFIK = traefik
HELM_REPO_URL_TRAEFIK = https://traefik.github.io/charts
TRAEFIK_RELEASE_NAME = traefik
TRAFIK_RELEASE_NAME = trafiks
NAMESPACE ?= default

# Install Traefik ingress controller
install-traefik:
	@echo "Adding Traefik Helm repository..."
	@helm repo add $(HELM_REPO_TRAEFIK) $(HELM_REPO_URL_TRAEFIK)
	@echo "Updating Helm repositories..."
	@helm repo update
	@echo "Installing Traefik..."
	@helm install $(TRAEFIK_RELEASE_NAME) $(HELM_REPO_TRAEFIK)/traefik

# Upgrade Traefik
upgrade-traefik:
	@echo "Updating Helm repositories..."
	@helm repo update
	@echo "Upgrading Traefik..."
	@helm upgrade $(TRAEFIK_RELEASE_NAME) $(HELM_REPO_TRAEFIK)/traefik

# Uninstall Traefik
uninstall-traefik:
	@echo "Uninstalling Traefik..."
	@helm uninstall $(TRAEFIK_RELEASE_NAME)

# Install Trafiks application
install:
	@echo "Installing Trafiks..."
	@helm install $(TRAFIK_RELEASE_NAME) . --namespace $(NAMESPACE) --create-namespace

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

# Show Traefik status
status-traefik:
	@echo "Traefik status:"
	@helm status $(TRAEFIK_RELEASE_NAME) || echo "Traefik not installed"

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

.PHONY: install-traefik upgrade-traefik uninstall-traefik install upgrade uninstall install-all upgrade-all status-traefik status status-all lint template values
