.PHONY: verify index install-hooks setup

verify:
	python3 scripts/verify.py

index:
	python3 scripts/build-index.py

install-hooks:
	@command -v gitleaks >/dev/null || { echo "gitleaks is required: https://github.com/gitleaks/gitleaks"; exit 1; }
	git config core.hooksPath .githooks
	@echo "Installed the pre-commit hook: gitleaks, then scripts/verify.py."

setup:
	python3 skills/setup-jstack/scripts/setup.py --agent auto
