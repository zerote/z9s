# ArgoCD demo apps

Sample manifests used to exercise the z9s ArgoCD integration. Each directory is
referenced by an ArgoCD `Application` (see `varios/scripts/argocd-test-apps.sh`)
and is crafted to land in a specific health state:

| Directory      | Resulting ArgoCD health |
|----------------|-------------------------|
| `degraded/`    | **Degraded** (invalid image -> ImagePullBackOff) |
| `progressing/` | **Progressing** (readiness probe never passes) |
| `suspended/`   | **Suspended** (CronJob with `suspend: true`) |

Healthy/Synced and OutOfSync states are sourced from the upstream
[argocd-example-apps](https://github.com/argoproj/argocd-example-apps) repo.

These are for local testing only (e.g. a Rancher Desktop / kind cluster). Do not
deploy them to real clusters.
