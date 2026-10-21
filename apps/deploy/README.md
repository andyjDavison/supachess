# Deploying Supachess

```
browser ──https──▶ <domain>            Vercel (apps/ui, static SPA)
        ──https/wss──▶ api.<domain>    AWS Lightsail: Caddy ─▶ api:8080 ─▶ postgres
```

Frontend and API must share a registrable domain. The session cookie is `SameSite=Lax`
and host-only on `api.<domain>`; on `*.vercel.app` it would be a third-party cookie and
login would break in Safari/Chrome.

## Pipeline

| Workflow | Runs on | Does |
| --- | --- | --- |
| `.github/workflows/ci.yml` | every PR (and called by deploy) | `go vet`, `go test -race`; UI tests, typecheck + build |
| `.github/workflows/deploy.yml` | push to `main`, manual dispatch | CI → API image → VM → UI to Vercel |

`deploy.yml` only ships the side that changed (`apps/api`, `apps/db`, `apps/deploy` → API;
`apps/ui` → UI). A manual run (Actions → deploy → Run workflow) ships both. The API job:

1. builds `ghcr.io/andyjdavison/supachess-api:{latest,<sha>}` (linux/amd64),
2. copies `docker-compose.prod.yml` and `caddy/Caddyfile` to `~/supachess/` on the VM,
3. runs `compose pull && up -d` pinned to that commit's SHA, reloads Caddy,
4. polls `https://api.<domain>/api/me` until it answers 401.

The UI job then runs `vercel pull` / `vercel build` / `vercel deploy --prebuilt --prod`.
Vercel's own Git deploys are switched off in `apps/ui/vercel.json`, so a red CI never ships.

**Restarting the api drops every running game's clock timer** (they're in memory, see
PLAN.md known gaps). Add yourself as a required reviewer on the `production` environment
if you want to approve each deploy when nobody is mid-game.

## One-time setup

### 1. Lightsail instance

1. Create an instance: Linux/Unix → Ubuntu LTS, at least the 1 GB RAM plan (Postgres + api
   + Caddy). Lightsail is x86_64, which is what CI builds.
2. Networking tab → **attach a static IP** (otherwise the IP changes on stop/start and DNS
   breaks), and add IPv4 firewall rules for HTTP 80 and HTTPS 443 (SSH 22 is there by default).
3. SSH in as `ubuntu`, install Docker Engine + the compose plugin
   (docs.docker.com/engine/install/ubuntu), then `sudo usermod -aG docker ubuntu` and log
   back in.
4. `mkdir ~/supachess`, copy `.env.example` there as `.env` and fill it in. `DOMAIN` is the
   apex (`example.com`), not `api.example.com`. CI copies the other files but never `.env`.
5. GHCR access: after the first image is pushed, either make the `supachess-api` package
   public (GitHub → Packages → Package settings), or run `docker login ghcr.io -u <user>`
   on the VM with a classic PAT that has `read:packages`.

### 2. DNS

- `A  api.<domain> → <static IP>`. Caddy can't get a cert until this resolves.
- Apex/`www` → Vercel, using the records Vercel shows under the project's Domains.

### 3. Deploy SSH key

Make a key used only by CI, on your laptop:

```
ssh-keygen -t ed25519 -f supachess-deploy -N "" -C "github-actions deploy"
ssh-copy-id -i supachess-deploy.pub ubuntu@<static IP>     # or append to ~/.ssh/authorized_keys
```

For `LIGHTSAIL_HOST_FINGERPRINT`, read the server's host key fingerprint **on the VM**, so
there's no doubt about which machine it came from:

```
ssh-keygen -lf /etc/ssh/ssh_host_ecdsa_key.pub | awk '{print $2}'   # → SHA256:...
```

Any of the host's key types works (ecdsa, ed25519, rsa): the deploy job scans the host and
trusts only the key whose fingerprint equals this secret. This is the server's *host* key,
not one of the keys in `~/.ssh/authorized_keys`.

`authorized_keys` takes one key per line. When appending the deploy key, make sure it
starts on a new line (`wc -l ~/.ssh/authorized_keys` should count each key).

### 4. Vercel project

1. Create a project for this repo. Leave **Root Directory blank**: the CLI runs from inside
   `apps/ui` already.
2. Settings → Environment Variables (Production):
   `VITE_API_URL=https://api.<domain>`, `VITE_WS_URL=wss://api.<domain>`.
   They're baked in at build time, so redeploy (manual run of `deploy`) after changing them.
3. Locally, in `apps/ui`: `npx vercel link`, then read `orgId` and `projectId` from
   `apps/ui/.vercel/project.json` (gitignored).
4. Account Settings → Tokens → create a token for GitHub Actions.

### 5. GitHub

Settings → Environments → New environment `production`, then add:

| Kind | Name | Value |
| --- | --- | --- |
| secret | `LIGHTSAIL_HOST` | static IP |
| secret | `LIGHTSAIL_USER` | `ubuntu` |
| secret | `LIGHTSAIL_SSH_KEY` | contents of `supachess-deploy` (private key) |
| secret | `LIGHTSAIL_HOST_FINGERPRINT` | the `SHA256:...` from step 3 |
| secret | `VERCEL_TOKEN` | token from Vercel step 4 |
| secret | `VERCEL_ORG_ID` | `orgId` |
| secret | `VERCEL_PROJECT_ID` | `projectId` |
| variable | `API_DOMAIN` | `api.<domain>` |

Then Settings → Branches → protect `main`: require a PR and the status checks
`api` and `ui` (they appear after `ci` has run once on a PR).

## Manual operations (on the VM)

```
cd ~/supachess
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f api      # expect "migrations applied"
```

Rollback: re-run an older `deploy` run from the Actions tab, or on the VM
`API_TAG=<older sha> docker compose -f docker-compose.prod.yml up -d`.

Build the image locally the same way CI does, from `apps/`:
`docker build -f api/Dockerfile -t supachess-api .`

## Smoke test

1. `curl -i https://api.<domain>/api/me` → `401`.
2. Register, log in, reload: still logged in (cookie persisted, `/api/me` 200).
3. Two browsers (different accounts) queue the same preset → game starts, moves sync,
   clocks run, game ends with the modal.
4. Browser console: no CORS errors, no WS schema-validation errors.
