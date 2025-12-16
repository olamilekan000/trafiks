# Trafiks UI

React dashboard for the Trafiks proxy service.

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev
```

The development server will run on `http://localhost:5173` (or the next available port).

## Building for Production

```bash
# Build the UI
npm run build
```

This will create a `dist` directory in `routes/dist` that contains the production-ready static files.

**Important:** The UI files are embedded directly into the Go binary at compile time using Go's `embed` package. The files are embedded from `routes/dist/`. After building the UI, you must rebuild the Go binary for the changes to take effect:

```bash
# Build UI and Go binary together
make build

# Or manually:
cd web/trafiks-ui && npm run build
go build -o trafiks ./main.go
```

## Integration with Go Server

The built UI is **embedded directly into the Go binary** at compile time using Go's `embed` package. This means:
- The dashboard is always available when you run the binary
- No need to deploy separate static files
- The binary is self-contained

The embedded UI is served by the Go server at:
- `/dashboard` - Main dashboard route
- `/dashboard/*` - All dashboard routes (handled by React Router)

## Notes

- The base path is set to `/dashboard/` in `vite.config.js`
- The build output goes to `routes/dist/` (relative to the routes package)
- **Important:** You must build the UI (`make build-ui`) before building the Go binary, as the files are embedded at compile time
- If you see a compile error about "no matching files", run `make build-ui` first to generate the `routes/dist` directory
- The embedded files are served from memory, so there's no filesystem dependency at runtime
