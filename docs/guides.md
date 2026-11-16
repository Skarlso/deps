# Everyday tasks

Recipes for the dependency work that comes up every week: see what is outdated, update one dependency, hold one back, cut PR noise. Each one is a command you run in the checkout, or a few lines of `.deps.yaml`.

Keys link into the [configuration reference](reference/configuration.md), flags into the [CLI reference](reference/cli.md). Workflows, tokens, automerge and the dashboard are in [Running deps in CI](ci.md); coming from Renovate, start with [Migrating from Renovate](migrating-from-renovate.md).

| Task | Do this |
|---|---|
| [See what is outdated](#see-what-is-outdated) | `deps plan` |
| [Update the checkout now](#update-the-checkout-now) | `deps apply` |
| [Update a single dependency](#update-a-single-dependency) | `deps apply --config` with a throwaway config |
| [Find out why a dependency is not updated](#find-out-why-a-dependency-is-not-updated) | `deps explain '<name-glob>'` |
| [Stop updating a dependency](#stop-updating-a-dependency) | rule with [`ignore: true`](reference/configuration.md#rulesignore) |
| [Stay below a major](#stay-below-a-major) | rule with [`allow`](reference/configuration.md#rulesallow) |
| [Take only security fixes](#take-only-security-fixes) | rule with [`updateTypes`](reference/configuration.md#rulesupdatetypes) + `ignore: true` |
| [Skip brand-new releases](#skip-brand-new-releases) | [`minimumReleaseAge`](reference/configuration.md#minimumreleaseage) |
| [Get fewer, bigger PRs](#get-fewer-bigger-prs) | [`group`](reference/configuration.md#rulesgroup) and [`groups`](reference/configuration.md#groups) |
| [Pin actions and images](#pin-actions-and-images) | [`pin: digest`](reference/configuration.md#pin) |
| [Refresh lockfiles](#refresh-lockfiles) | [`lockFileMaintenance`](reference/configuration.md#lockfilemaintenance) |
| [Track a version deps does not see](#track-a-version-deps-does-not-see) | a `# deps:` marker and [`custom`](reference/configuration.md#custom) |
| [Try a config change before pushing it](#try-a-config-change-before-pushing-it) | `deps config check`, `deps plan --config`, `deps run --dry-run` |
| [Work on many repositories](#work-on-many-repositories) | `--repositories 'my-org/*'` |
| [Unstick an update PR](#unstick-an-update-pr) | `rebase` / `stop-updating` labels, dashboard checkboxes |
| [Keep deps itself current](#keep-deps-itself-current) | deps updates its own action pin and `extends` refs |

## See what is outdated

No config and no token needed; nothing is written.

```sh
deps plan                 # summary and the files that would change
deps plan --diff          # with the unified diff on a terminal
deps plan --show-skipped  # plus one line per skipped dependency, with the reason
```

The summary names every PR [`deps run`](reference/cli.md#deps-run) would open:

```
Summary: 4 dependencies, 6 updates in 6 PRs, 0 skipped, 0 errors (680ms)
  PR deps/actions-checkout  chore(deps): update actions/checkout to v4.4.0
      actions/checkout  .github/workflows/ci.yml:7  v4.1.0 -> v4.4.0  (minor)
```

Piped or redirected stdout always carries the diff, so `deps plan > deps.diff && git apply --check deps.diff` checks it. Exit codes: [`deps help exit-codes`](reference/cli.md#exit-codes).

## Update the checkout now

Before a release, or to get a dirty tree you can test:

```sh
deps apply && git diff --stat
```

`apply` writes the manifests and refreshes the lockfiles of the ecosystems it manages (`go mod tidy`, `package-lock.json`, `Cargo.lock`, `uv.lock`, …; the full list is in [`deps apply`](reference/cli.md#deps-apply)). Nothing is committed or pushed. A dependency with updates in several PR buckets gets the newest non-major one here, and its major is reported with `, separate PR`.

## Update a single dependency

There is no name filter on `apply`; narrow the policy instead, with a config outside the repository so the committed one stays untouched:

```yaml
# /tmp/one-dep.yaml
version: 1
rules:
  - ignore: true              # ignore everything ...
  - names: ['alpine']         # ... except this dependency (doublestar globs, as in rules[].names)
    ignore: false
```

```sh
deps plan  --config /tmp/one-dep.yaml   # check it
deps apply --config /tmp/one-dep.yaml   # write it
```

```
Summary: 4 dependencies, 1 update in 1 PR, 3 skipped, 0 errors
  PR deps/alpine  chore(deps): update alpine to 3.24
      alpine  Dockerfile:1  3.19 -> 3.24  (minor)
```

## Find out why a dependency is not updated

[`deps explain`](reference/cli.md#deps-explain) resolves only the names you give it and prints the rules that matched, the effective policy, every rejected version by reason, and the result:

```sh
deps explain 'actions/*'
deps explain 'k8s.io/**' github.com/spf13/cobra
```

```
actions/checkout  .github/workflows/ci.yml:7  (github-actions, action, git-tags, tag)
  current   v4.1.0
  rules     #1 match: names=[actions/checkout]; apply: allow=<7 (from .deps.yaml:5)
  policy    pin=keep allow=<7 allowedUpdateTypes=major,minor,patch,digest,pin group=- schedule=always minAge=-
  candidates 68
  rejected  not newer: 35 (v4.1.0, ...)  ignored by rule #0 for major: 10 (v7.0.1, ...)  precision: 7  invalid: 2
  result    update v4.1.0 -> v4.4.0 (minor)
```

`rules` names the file and line of each matching rule, so a surprising policy points at the line that caused it. A dependency that does not appear at all was never extracted: `deps plan --show-skipped` prints its skip reason (`ignored by default (indirect; enable with a rule setting ignore: false)`, an unsupported file, …), and the manager's section in [managers and datasources](reference/managers.md) lists every reason it can report.

## Stop updating a dependency

```yaml
rules:
  - names: ['actions/*']   # or managers: [...], paths: ['charts/**'], datasources: [...]
    ignore: true
```

`deps plan --show-skipped` then reports `ignored by rule #0` per occurrence. Rules are read top to bottom and later rules override single fields, so a broad `ignore: true` can be narrowed again by a later rule with `ignore: false` (that is also how you opt *in* default-ignored dep types: indirect Go requirements, npm `peerDependencies`, submodules — see [`rules[].ignore`](reference/configuration.md#rulesignore)).

A dependency ignored this way gets no security updates either; for those see [below](#take-only-security-fixes).

## Stay below a major

Two ways, depending on whether the limit is one dependency or the whole repository:

```yaml
rules:
  - names: ['actions/checkout']
    allow: '< 7'               # semver constraint, or Renovate's regex form /^\d+\.\d+$/
  - updateTypes: [major]       # no major PRs at all
    ignore: true
```

Majors rejected this way show in `deps explain` as `ignored by rule #0 for major` / `outside allow <7`. Majors that *are* allowed get their own PR per newer major ([`separate`](reference/configuration.md#separate)); closing such a PR unmerged makes deps leave that major alone until the next one appears.

## Take only security fixes

Security updates bypass `allow`, `allowedUpdateTypes`, schedules, `minimumReleaseAge` and rules that carry `updateTypes` — but never a plain `ignore: true`. So an `ignore` with `updateTypes` freezes a dependency except for advisories:

```yaml
rules:
  - names: ['github.com/go-jose/**']        # leave out names: for the whole repository
    updateTypes: [major, minor, patch, digest, pin]
    ignore: true
```

```
update  go.mod:6  github.com/go-jose/go-jose/v4  v4.0.1 -> v4.1.6  (minor)  security: GO-2025-3485,… (high)
  PR deps/github.com-go-jose-go-jose-v4  fix(deps): update github.com/go-jose/go-jose/v4 to v4.1.6 [security]
```

Advisories come from OSV and carry CVSS, EPSS and CISA KEV data in the PR; [`builtins.security`](reference/configuration.md#builtinssecurity) lists the ecosystems and turns them off.

## Skip brand-new releases

Give a release time to be withdrawn before deps proposes it:

```yaml
minimumReleaseAge: 3 days
rules:
  - datasources: [npm]
    minimumReleaseAge: 7 days
```

Versions held back are reported by `deps explain` as `too new (released …, minimum 3 days)`, and the dashboard lists them under the releases it holds back. A datasource without release timestamps blocks the update by default — [`minimumReleaseAgeBehaviour`](reference/configuration.md#minimumreleaseagebehaviour) relaxes that. Security updates ignore the delay.

## Get fewer, bigger PRs

One PR for a set of dependencies, with the PR's settings written once:

```yaml
rules:
  - managers: [github-actions]
    updateTypes: [minor, patch]
    group: actions
groups:
  actions:
    automerge: true
    schedule: ['0 5 * * 1']     # a cron of the workflow
```

```
  PR deps/group/actions  chore(deps): update actions (2 dependencies)  [automerge]
      actions/checkout  .github/workflows/ci.yml:7  v4.1.0 -> v4.4.0  (minor)
      actions/setup-go  .github/workflows/ci.yml:8  v5.0.0 -> v5.6.0  (minor)
```

Related knobs: [`separate`](reference/configuration.md#separate) (how one dependency's candidates split into PRs), [`prConcurrentLimit`](reference/configuration.md#prconcurrentlimit) and [`prHourlyLimit`](reference/configuration.md#prhourlylimit) (how many PRs are open or created), [`schedule`](reference/configuration.md#schedule) (when new PRs appear), [`dashboardApproval`](reference/configuration.md#dashboardapproval) (nothing is opened until ticked on the dashboard). Monorepo members are grouped already — [`builtins.monorepos`](reference/configuration.md#builtinsmonorepos).

## Pin actions and images

```yaml
pin: digest                   # actions to SHAs, images to digests
rules:
  - managers: [github-actions]
    pin: keep                 # ... except these
```

With `pin: digest` an action becomes `actions/checkout@11d5960a… # v4.4.0` and an image `alpine:3.24@sha256:f59abe…`, so the readable version stays next to the digest. Version *ranges* (npm, cargo, Python, Composer, …) are a separate decision: [`rangeStrategy`](reference/configuration.md#rangestrategy) chooses between widening the range and bumping it.

## Refresh lockfiles

Manifest updates keep their lockfile in sync automatically. To also move transitive dependencies that no manifest change touches:

```yaml
lockFileMaintenance:
  enabled: true
  schedule: ['0 5 * * 1']     # a cron of the workflow; without one, every run
```

`deps run` then keeps one extra PR on `<branchPrefix>lock-file-maintenance` that regenerates every lockfile it manages (and the `// indirect` block of `go.mod` with `go.sum`), and closes it when the lockfiles are current. Locally the same refresh is `deps apply` plus the ecosystem's own update command.

## Track a version deps does not see

Any version pinned in a file deps has no manager for — a Makefile variable, an `.env`, a script — becomes a dependency with a marker above it:

```make
# deps: github-releases kubernetes-sigs/kustomize
KUSTOMIZE_VERSION ?= v5.4.1
```

```yaml
custom:
  - files: ['Makefile']       # files the marker manager scans
rules:
  - names: ['kubernetes-sigs/kustomize']
    extractVersion: '^kustomize/(?<version>.+)$'   # tags are kustomize/v5.8.3
```

```sh
deps explain 'kubernetes-sigs/*'
# kubernetes-sigs/kustomize  Makefile:2  (custom, annotation, github-releases, semver)
#   result    update v5.4.1 -> v5.8.3 (minor)
```

Renovate's `# renovate: datasource=… depName=…` markers are read as they are. Without a marker, a [`custom[].match`](reference/configuration.md#custommatch) regex or a [`custom[].jsonata`](reference/configuration.md#customjsonata) query extracts the version instead, and [`custom[].checksum`](reference/configuration.md#customchecksum) updates a checksum next to it.

## Try a config change before pushing it

```sh
deps config check                     # strict validation, offline, exit 2 on errors
deps config show --sources            # effective values, each with default | file | env | flag
deps plan --config /tmp/try.yaml      # what the change would do, without touching .deps.yaml
deps run --dry-run --show-skipped     # the PRs a real run would create, update or close
```

`config check` reports every problem with position and a suggestion, which is what makes it a good pre-commit hook ([`deps config check`](reference/cli.md#deps-config-check) has the hook):

```
.deps.yaml:2:1: automerg: unknown key (did you mean automerge?)
.deps.yaml:4:5: rules[0].matchPackageNames: unknown key (Renovate's matchPackageNames is names here)
```

`deps run --dry-run` works without a token. To see the effect of a config change on a pull request automatically, add the [preview job](ci.md#preview-on-pull-requests); `--trigger-schedule` and `--schedule-time` replay a specific cron or clock time.

## Work on many repositories

Every read-only command takes `--repositories`; deps clones each repository itself, so no checkout is needed. A token (`DEPS_TOKEN`, `GITHUB_TOKEN`, …) is required to list and clone them:

```sh
deps plan --repositories 'my-org/*'                      # audit an organisation, read-only
deps explain --repositories 'my-org/*' 'golang.org/x/**' # one dependency across it
deps apply --repositories 'my-org/svc-*' --into work     # clones kept in work/<owner>/<name>
```

`deps run --repositories` serves many repositories from one job; shared rules belong in an [`extends`](reference/configuration.md#extends) config. Both are in [Shared config and many repositories](ci.md#shared-config-and-many-repositories).

## Unstick an update PR

| Situation | What to do |
|---|---|
| PR conflicts or its base moved | tick its rebase checkbox, or add the `rebase` label ([`rebaseLabel`](reference/configuration.md#rebaselabel)); the next run rebuilds the branch and removes the label |
| You are editing the PR yourself | add the `stop-updating` label ([`stopUpdatingLabel`](reference/configuration.md#stopupdatinglabel)): deps leaves it alone, and does not close it as obsolete |
| `skipped: branch modified by someone else` | a rebase request discards the foreign commits; to keep them, list their author in [`git.ignoredAuthors`](reference/configuration.md#gitignoredauthors) |
| `skipped: closed PR for this version` / `for this major` | you closed that PR unmerged; tick *recreate* on the dashboard to get it back, or add an `ignore` rule to drop it for good |
| PR waits for approval or its schedule | tick it on the dashboard: that run creates it, regardless of schedule or limits |

Every status a run can print is listed under [`deps run`](reference/cli.md#deps-run), and the dashboard in [Dependency dashboard](ci.md#dependency-dashboard).

## Keep deps itself current

- The workflow's `uses: jakobmoellerdev/deps@vX.Y.Z` is an ordinary actions dependency, so deps proposes its own updates.
- `extends` entries with a `#<ref>` are updated by the [`deps-config`](reference/managers.md#deps-config) manager, so a shared organisation config moves with its tags.
- The `# yaml-language-server: $schema=…/vX.Y.Z/deps.schema.json` modeline is editor metadata and is not updated for you: bump it with the version, or point it at a local `deps schema > deps.schema.json` ([editor support](reference/configuration.md)).
