default: deps prod

deps:
	NO_UPDATE_NOTIFIER=true npm install --no-fund --no-audit

prod:
	npx vite build --emptyOutDir

dev:
	npx vite preview
