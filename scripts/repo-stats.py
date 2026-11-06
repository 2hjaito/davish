#!/usr/bin/env python3
"""Appends the repo's public numbers to CSV files, one row per day.

GitHub keeps traffic for only 14 days and download counts only as a running
total, so history exists only if it is saved. Usage: repo-stats.py <folder>.

GITHUB_TOKEN reads the public numbers. TRAFFIC_TOKEN (optional) also reads
views, clones and referrers, which need access the Actions token does not have.
"""
import csv
import datetime
import json
import os
import sys
import urllib.error
import urllib.request

API = os.environ.get("GITHUB_API_URL", "https://api.github.com")
REPO = os.environ.get("GITHUB_REPOSITORY", "2hjaito/davish")

DAILY = ["date", "stars", "forks", "watchers", "installers", "macos", "windows", "linux", "checksums"]
TRAFFIC = ["date", "views", "unique_views", "clones", "unique_clones"]
REFERRERS = ["date", "referrer", "views", "unique_views"]


def get(path, token):
    req = urllib.request.Request(API + path)
    req.add_header("Accept", "application/vnd.github+json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    with urllib.request.urlopen(req, timeout=30) as res:
        return json.load(res)


def releases(token):
    page = 1
    while True:
        batch = get(f"/repos/{REPO}/releases?per_page=100&page={page}", token)
        yield from batch
        if len(batch) < 100:
            return
        page += 1


def platform(name):
    name = name.lower()
    if "macos" in name:
        return "macos"
    if "windows" in name:
        return "windows"
    return "linux" if "linux" in name else None


def daily(today, token):
    repo = get(f"/repos/{REPO}", token)
    row = {
        "date": today,
        "stars": repo["stargazers_count"],
        "forks": repo["forks_count"],
        "watchers": repo["subscribers_count"],
        "installers": 0,
        "macos": 0,
        "windows": 0,
        "linux": 0,
        "checksums": 0,
    }
    for release in releases(token):
        for asset in release["assets"]:
            count = asset["download_count"]
            target = platform(asset["name"])
            if target:
                row[target] += count
                row["installers"] += count
            elif asset["name"].startswith("SHA256SUMS"):
                row["checksums"] += count
    return row


def merge(path, fields, rows, key):
    """Writes rows over the file's, one per key: the newest numbers for a day win."""
    merged = {}
    if os.path.exists(path):
        with open(path, newline="", encoding="utf-8") as f:
            merged = {key(r): r for r in csv.DictReader(f)}
    merged.update({key(r): r for r in rows})
    with open(path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fields, lineterminator="\n")
        writer.writeheader()
        writer.writerows(merged[k] for k in sorted(merged))


def traffic(folder, today, token):
    views = get(f"/repos/{REPO}/traffic/views", token)["views"]
    clones = get(f"/repos/{REPO}/traffic/clones", token)["clones"]
    days = {}
    for v in views:
        day = v["timestamp"][:10]
        days[day] = {"date": day, "views": v["count"], "unique_views": v["uniques"], "clones": 0, "unique_clones": 0}
    for c in clones:
        day = c["timestamp"][:10]
        row = days.setdefault(day, {"date": day, "views": 0, "unique_views": 0})
        row.update(clones=c["count"], unique_clones=c["uniques"])
    merge(os.path.join(folder, "traffic.csv"), TRAFFIC, days.values(), lambda r: r["date"])

    refs = get(f"/repos/{REPO}/traffic/popular/referrers", token)
    rows = [{"date": today, "referrer": r["referrer"], "views": r["count"], "unique_views": r["uniques"]} for r in refs]
    merge(os.path.join(folder, "referrers.csv"), REFERRERS, rows, lambda r: (r["date"], r["referrer"]))


def main():
    folder = sys.argv[1]
    today = datetime.datetime.now(datetime.timezone.utc).date().isoformat()
    row = daily(today, os.environ.get("GITHUB_TOKEN"))
    merge(os.path.join(folder, "daily.csv"), DAILY, [row], lambda r: r["date"])
    print(row)

    token = os.environ.get("TRAFFIC_TOKEN")
    if not token:
        print("::notice::TRAFFIC_TOKEN is not set: skipping views, clones and referrers")
        return
    try:
        traffic(folder, today, token)
    except urllib.error.HTTPError as e:
        # The public numbers are saved already; a bad token must not lose them.
        print(f"::warning::traffic not read: HTTP {e.code}")


if __name__ == "__main__":
    main()
