# Runwell plugin template

The starting point for a [Runwell](https://github.com/Martin-Business-Consultants/runwellgo)
plugin in its own repository. It holds **Hello** (key `hello`), the smallest plugin and the one
[docs/plugins.md](https://github.com/Martin-Business-Consultants/runwellgo/blob/main/docs/plugins.md)
walks through: a note on every client's page (a panel), a page in the sidebar with a small form
kept in its own storage, and an ear for `client.created`. It asks for:

- `storage`: keeps values in its own key-value namespace
- `records:read`: reads clients, engagements, work and the other core records, as agents see them

Hello is in the official Runwell binary as the example plugin; an owner can add it from
Settings › Plugins.

## Making your own

1. **Use this template** on GitHub (or copy it), and change the module path in `go.mod` to your
   repository's.
2. Rename the package and give the manifest your own `Key` (lowercase, `a-z0-9_`; it names your
   routes, `/api/x/<key>/…`, your storage and your tables, `<key>_*`, so it never changes once
   people install it), `Name`, `Version` and `Description`.
3. Ask for only the grants you need (`Permissions`). Each is a capability of `*plugin.API`; a
   call without its grant fails. The list is in package `plugin`'s documentation.
4. Write handlers (`func(ctx, *plugin.API, *plugin.Request) (*plugin.Response, error)`) for the
   routes you declare, and describe your screens in the manifest (`Screens`, `Panels`, `Nav`,
   `Settings`, `QuickActions`, `Tabs`): the app draws them with its own components.
5. Test it with [`plugintest`](https://pkg.go.dev/github.com/Martin-Business-Consultants/runwellgo/v3/plugintest),
   as [hello_test.go](hello_test.go) does: a host on a fresh database, with a stand-in for the
   core that you extend to answer what a test needs.
6. Tag a release (`v1.0.0`) when it's ready, matching the manifest's `Version`.

Use only `github.com/Martin-Business-Consultants/runwellgo/v3/plugin` (and `plugintest` in
tests). That's the versioned contract: within Runwell v3 it only grows. Go doesn't stop a
plugin importing `database/sql` or `os`; don't, so that the grants stay what an owner reads.

## Running Runwell with your plugin

Runwell is a library, like PocketBase: a binary is a `main.go` that hands it its plugins. Make a
repository for your Runwell (or a `cmd/` in this one) with:

```go
package main

import (
	"os"

	ai "github.com/Martin-Business-Consultants/runwellgo-ai"
	"github.com/Martin-Business-Consultants/runwellgo/v3/runwell"
	hello "github.com/Martin-Business-Consultants/runwellgo-plugin-template"
)

// version is set from the release tag: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	os.Exit(runwell.Main(version, hello.Module(), ai.Module()))
}
```

`go mod init`, `go get` the modules, and `go build -o runwell .`. List every plugin you want,
the official ones included (runwellgo's `cmd/runwell/main.go` lists them): the binary has these
and no others. To deploy it to Hoster, use runwellgo's `Dockerfile` and `config/deploy.yml`, with
the build step pointed at your main package:

```dockerfile
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${V#v}" -o /out/runwell .
```

The app your binary serves is the one embedded in the runwellgo module, which carries the
official plugins' Svelte screens. A plugin in its own repository draws its screens through its
manifest; one that needs Svelte screens of its own is added to the official build (runwellgo's
`cmd/runwell`), whose `make plugin-ui` copies each plugin module's `ui/` into the app.

## Developing

```sh
go vet ./... && go test ./...
golangci-lint run ./...   # v2.15.0, .golangci.yml
```

## License

[FSL-1.1-MIT](LICENSE.md), like Runwell.
