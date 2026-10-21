# Marketing screenshots

The PNGs in this directory are the screenshots used in the project README and
docs. They are generated, so don't edit them by hand. To regenerate them:

```bash
make screenshots          # from the repo root
make -C var/marketing/screenshots
```

`make` builds Japella, starts a **throwaway stack**, seeds it with fictional
data, captures every page listed in `screenshots.ini`, writes
`screenshot-manifest.yaml`, copies the images into the docs, and then tears the
stack down again, even if a step fails.

## Requirements

- `podman` (or `CONTAINER=docker make`)
- Go and Node/npm, to build the service and frontend
- `repo-helper`, Chrome/Chromium and a matching `chromedriver`
- `curl`, `jq` and `python3`

## What runs

| Piece | Where | Notes |
|-------|-------|-------|
| MariaDB | container `japella-mkt-mariadb`, `127.0.0.1:13307` | No volume; deleted by `make down` |
| Japella | `127.0.0.1:18380`, from `service/japella` and `frontend/dist` | `HOME`, logs and PID live in `tmp/` |

The backend runs with `HOME=tmp/home`, so it never reads a developer's
`~/.config/japella/config.yaml`, and with its outbound HTTP pointed at a closed
local port, so the demo accounts can never reach a real social network. The
backend, database and browser all run in UTC.

## Data

Everything is fictional: "Acme Roasters", a made-up coffee brand, with
`example.org`/`example.com` accounts and the users `admin`, `jordan` and `sam`
(throwaway passwords that exist only in the temporary database).

- `seed.sh` seeds through the Connect API: settings, users, the Marketing
  group and its account shares, campaigns, canned posts, scheduled posts,
  drafts, the "Brand review" approval policy, and posts waiting for approval.
- `seed/accounts.sql`, `seed/connectors.sql` and `seed/history.sql` cover what
  the API can't create without real OAuth: the Mastodon/Bluesky/X/Facebook/
  Instagram accounts, placeholder OAuth client ids, publishing history and
  feed items. Dates are relative to today, so the calendar always has content.

## Browsing the demo stack

```bash
make up seed     # then open http://127.0.0.1:18380/ and log in as admin/admin
make down
```

## Adding a screenshot

Add a section to `screenshots.ini` with `url`, `title` and `description` (and
`theme = dark` for dark variants, which go last), then run `make`. The manifest
generator fails if a section is missing its text or PNG.
