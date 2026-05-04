// Package webui exposes the production build of the Vite + React management
// UI (compiled into webui/dist) as an embedded filesystem so the Go binary
// can serve `/` without any external assets.
//
// During `make webui-dev` the Vite dev server runs on :5173 and proxies
// `/api`, `/swagger`, `/scalar`, `/healthz` and `/readyz` to the Go backend
// on :8081, so this embedded filesystem is unused in dev. The embedded
// dist/ folder always contains a placeholder file (.gitkeep) so go:embed
// succeeds even on a fresh checkout where `npm run build` hasn't run yet.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the rooted dist/ filesystem. Use the returned fs.FS as the
// source for fiber's filesystem middleware. The returned filesystem is
// guaranteed to be non-nil; if no built assets are present it simply
// contains the .gitkeep placeholder.
func FS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// embed.FS guarantees the directory exists at compile time, so
		// this can only happen if someone removes the //go:embed directive.
		panic("webui: embedded dist subtree missing: " + err.Error())
	}
	return sub
}

// HasIndex reports whether a real Vite build is embedded (ie. dist/index.html
// exists). Routes use this to decide whether to mount the SPA fallback or
// fall back to a tiny placeholder page that nudges devs to run the build.
func HasIndex() bool {
	_, err := fs.Stat(FS(), "index.html")
	return err == nil
}
