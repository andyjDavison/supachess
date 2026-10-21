# Deploying Supachess

```
browser ──https──▶ <domain>            Vercel (apps/ui, static SPA)
        ──https/wss──▶ api.<domain>    Oracle VM: Caddy ─▶ api:8080 ─▶ postgres
```

Frontend and API must share a registrable domain. The session cookie is `SameSite=Lax`
and host-only on `api.<domain>`; on `*.vercel.app` it would be a third-party cookie and
login would break in Safari/Chrome.

## Images

`.github/workflows/api-image.yml` runs on every push to `main` that touches `apps/api` or
`apps/db`: vet + tests, then a multi-arch build pushed to
`ghcr.io/andyjdavison/supachess-api:{latest,<sha>}`. Migrations are baked into the image
and applied when the api container starts.

Build it locally the same way, from `apps/`:

```
docker build -f api/Dockerfile -t supachess-api .
```

## One-time VM setup (Ubuntu, Ampere/arm64)

1. Install Docker Engine and the compose plugin (docs.docker.com/engine/install/ubuntu).
2. Open 80 and 443 (TCP, plus 443/UDP for HTTP/3) in **both** places — Oracle blocks them twice:
   - OCI console → the VM's VCN → Security List → ingress rules from `0.0.0.0/0`.
   - On the VM, Oracle's Ubuntu image ships iptables rules that reject everything but SSH:
     ```
     sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 80 -j ACCEPT
     sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 443 -j ACCEPT
     sudo iptables -I INPUT 6 -m state --state NEW -p udp --dport 443 -j ACCEPT
     sudo netfilter-persistent save
     ```
3. DNS: `A  api.<domain> → <VM public IP>`. Caddy can't get a cert until this resolves.
4. GHCR access: either make the `supachess-api` package public (GitHub → Packages →
   settings), or `docker login ghcr.io -u <user>` with a PAT that has `read:packages`.
5. Copy this directory to the VM (e.g. `~/supachess/`), then `cp .env.example .env` and
   fill it in. `DOMAIN` is the apex (`example.com`), not `api.example.com`.

## Deploy / update

```
cd ~/supachess
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
docker compose -f docker-compose.prod.yml logs -f api   # expect "migrations applied"
```

Rollback: set `API_TAG=<previous sha>` in `.env` and run the two commands again.

**Restarting the api drops every running game's clock timer** (they're in memory, see
PLAN.md known gaps). Deploy when nobody is mid-game.

## Vercel (frontend)

1. Import the repo. Root directory `apps/ui`, framework preset Vite (build `pnpm run build`,
   output `dist`).
2. Environment variables (Production):
   - `VITE_API_URL=https://api.<domain>`
   - `VITE_WS_URL=wss://api.<domain>`

   These are baked in at build time — redeploy after changing them.
3. Add `<domain>` (and `www` if wanted) under Domains and set the DNS records Vercel shows.
   `vercel.json` rewrites every path to `index.html` so deep links like `/game/<id>` load.

## Smoke test

1. `curl -i https://api.<domain>/api/me` → `401`.
2. Register, log in, reload: still logged in (cookie persisted, `/api/me` 200).
3. Two browsers (different accounts) queue the same preset → game starts, moves sync,
   clocks run, game ends with the modal.
4. Browser console: no CORS errors, no WS schema-validation errors.
