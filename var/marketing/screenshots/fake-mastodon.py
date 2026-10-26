#!/usr/bin/env python3
"""A tiny, fixed stand-in for a Mastodon instance, for the feed's search columns.

Serves only the two endpoints search columns use, from seed/fake-mastodon.json:
  GET /api/v1/timelines/tag/<tag>   (hashtag columns)
  GET /api/v2/search?q=<query>      (search columns)
Everything else is a 404, so feed and engagement fetches still find nothing.
Posts are dated relative to now, so the columns always look current.

Usage: fake-mastodon.py <port>
"""
import json
import sys
from datetime import datetime, timedelta, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, unquote, urlparse

FIXTURES = json.loads((Path(__file__).resolve().parent / "seed" / "fake-mastodon.json").read_text())


def status(post):
    local, _, instance = post["acct"].partition("@")
    created = datetime.now(timezone.utc) - timedelta(minutes=post["minutes_ago"])
    profile = "https://%s/@%s" % (instance, local)
    return {
        "id": post["id"],
        "uri": "%s/%s" % (profile, post["id"]),
        "content": post["content"],
        "created_at": created.strftime("%Y-%m-%dT%H:%M:%S.000Z"),
        "language": "en",
        "favourited": False,
        "favourites_count": post["favourites"],
        "replies_count": post["replies"],
        "account": {
            "id": "fake-" + local,
            "username": local,
            "acct": post["acct"],
            "avatar": "",
            "uri": profile,
            "url": profile,
        },
    }


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        url = urlparse(self.path)
        body = None
        if url.path.startswith("/api/v1/timelines/tag/"):
            tag = unquote(url.path.rsplit("/", 1)[-1]).lower()
            body = [status(p) for p in FIXTURES["tags"].get(tag, [])]
        elif url.path == "/api/v2/search":
            q = parse_qs(url.query).get("q", [""])[0].strip().lower()
            body = {"accounts": [], "hashtags": [], "statuses": [status(p) for p in FIXTURES["search"].get(q, [])]}
        if body is None:
            self.send_error(404)
            return
        data = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, fmt, *args):
        sys.stderr.write("fake-mastodon: " + fmt % args + "\n")


if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
