# Benchmarks

`deps plan` (scan, all registry lookups, patched diff) on large open-source repositories, limited to 4 CPUs (`taskset -c 0-3`, `GOMAXPROCS=4`) like a standard CI runner, median of 7 runs. The README shows a shortened version of this table.

| Repository | Ecosystems | Tracked files | Updates found | Wall time | CPU time | Peak memory |
|---|---|--:|--:|--:|--:|--:|
| openclaw/openclaw 🦞 | pnpm, Swift, Ruby, Actions | 65,530 | 702 | 2.7 s | 1.4 s | 119 MB |
| vercel/next.js | pnpm, Rust, Actions | 35,112 | 1,619 | 2.2 s | 1.8 s | 148 MB |
| kubernetes/kubernetes | Go, containers | 31,988 | 153 | 1.5 s | 0.6 s | 69 MB |
| grafana/grafana | Go, Yarn, containers, Actions | 24,654 | 1,036 | 13.4 s¹ | 1.7 s | 122 MB |
| backstage/backstage | Yarn workspaces | 12,709 | 1,035 | 2.6 s | 1.5 s | 135 MB |
| spring-projects/spring-boot | Gradle, Maven | 11,994 | 41 | 1.0 s | 0.4 s | 43 MB |
| astral-sh/ruff | Cargo, Python, npm | 11,325 | 152 | 2.1 s | 0.7 s | 70 MB |
| can1357/oh-my-pi | Bun, Rust, Python | 10,646 | 194 | 1.9 s | 0.7 s | 63 MB |
| istio/istio | Go, containers, Helm | 6,798 | 131 | 5.3 s | 1.2 s | 68 MB |
| argoproj/argo-cd | Go, Yarn, containers | 6,262 | 242 | 4.4 s | 0.9 s | 93 MB |
| open-component-model | Go, containers, Helm | 3,148 | 40 | 3.6 s | 0.5 s | 54 MB |
| kubernetes-sigs/cluster-api | Go, containers | 2,401 | 33 | 0.8 s | 0.3 s | 50 MB |

CPU time is deps itself; the rest of the wall time is registry latency, mostly container registries paging through tag lists.

Peak memory is the noisiest column: it is set by how much registry data arrives between two garbage collections, which is network timing rather than anything the repository decides. Over seven runs of the same binary on the same checkout, backstage ranged 102–150 MB and next.js 144–159 MB, while the small rows (spring-boot 42–43 MB, oh-my-pi 62–64 MB) barely move. Read the column as a band, not a figure: a difference under about 40 % on the large repositories says nothing.

¹ `proxy.golang.org` does not answer the probe for a newer major module path of a few modules (grafana: `github.com/aws/aws-sdk-go-v2/service/kms/v2`), so those lookups hit their 10 s timeout in most runs; without them the run takes about 3 s. Which modules these are changes over time, and argo-cd (`github.com/golang/groupcache/v2`) only hit it in one run of seven.

`deps run` adds the lockfile tools (`go mod tidy`, npm/pnpm/yarn, uv, cargo …) for the PRs it updates; PRs whose content didn't change are not pushed again.

## How the numbers are produced

1. Clone each repository (shallow is enough) under a scratch directory.
2. For every repository, run `taskset -c 0-3 env GOMAXPROCS=4 /usr/bin/time -v deps plan --color never` seven times warm and take the median wall time, user+sys CPU time and maximum resident set size. Seven rather than three because peak memory swings by up to half its value between runs; three samples of that distribution look like a regression that is not there.
3. Tracked files is `git ls-files | wc -l` of the checkout, updates found is the number of updates in the plan.

Registry latency dominates most rows, so re-measure all repositories in one session; numbers from different days are not comparable. Compare CPU time, not wall time or peak memory, when looking for a change in deps itself.
