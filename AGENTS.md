# AGENTS.md

An OpenBao secrets engine that serves 1Password items live through the
1Password service-account SDK, from an in-memory replica that is never
written to OpenBao storage. Forked from NavistAu/openbao-plugin-secrets-onepassword
at v0.1.0. `backend/` is the engine, `cmd/` the plugin process.

## Working

- Every tool comes from mise, and every repeatable action is a mise task
  (`mise tasks` lists them). Add a task for a new workflow rather than a script.
- Changes are hard cutovers: rename, remove and migrate in one change, with no
  compatibility shims, aliases or deprecated paths left behind.
- Tests are offline and run against the fakes in `backend/opclient_fake.go`
  and OpenBao's in-memory storage. Never add a test that needs a live
  1Password account.
- Secret values never reach logs, errors, `op/status` or config reads. The
  service-account token is concealed on read and passed to child processes
  only through the environment.
- A change to what the replica caches, for how long, or under what staleness
  guarantees is called out in the commit message and the release notes.
- Paths and response shapes are documented in the README's Paths table in the
  same commit that changes them.

## Releases

Pushing a `vX.Y.Z` tag runs `.github/workflows/release.yml`: reproducible
binaries from `mise run release:build`, the OCI image from `Containerfile`,
attestations, the GitHub release, and communique notes. The plugin reports
the tag as its version, and OpenBao rejects a registration whose version
differs.

## Done

A change is done when the **CI result** check on GitHub is green after pushing.
Run `mise run ci` locally first. Commits use conventional commit messages
(`feat(kv): …`, `fix(replica): …`).
