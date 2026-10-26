#!/usr/bin/env bash
# Seeds the throwaway backend with fictional data, mostly via its public
# Connect API. SQL is only used for state the API can't create (see seed/*.sql).
#
# Usage: seed.sh <backend-base-url> <tmp-dir>
# Env:   SQL  command that reads SQL on stdin and runs it against the demo DB
set -euo pipefail

base="${1:?backend base url}"
tmp="${2:?tmp dir}"
: "${SQL:?SQL must be set to a command that runs SQL from stdin}"
here="$(cd "$(dirname "$0")" && pwd)"
seed="$here/seed"
api="$base/api/japella.controlapi.v1.JapellaControlApiService"

# rpc <cookie-jar> <method> <json>; prints the JSON response.
rpc() {
	curl -fsS -b "$1" -c "$1" -X POST -H 'Content-Type: application/json' -d "$3" "$api/$2"
}

login() {
	rpc "$tmp/$1.cookies" LoginWithUsernameAndPassword \
		"$(jq -nc --arg u "$1" --arg p "$2" '{username: $u, password: $p}')" >/dev/null
}

# "YYYY-MM-DDTHH:MM", <days> from today at <time>; the backend runs in UTC.
when() {
	echo "$(TZ=UTC date -d "+$1 days" +%Y-%m-%d)T$2"
}

admin="$tmp/admin.cookies"
rm -f "$tmp"/*.cookies
login admin admin

echo "Seeding settings"
jq -c 'to_entries[] | {keyName: .key, valueInt: .value}' "$seed/cvars.json" | while read -r kv; do
	rpc "$admin" SetCvar "$kv" >/dev/null
done
# Starts the OAuth connectors so accounts show their network's icon. They
# only touch the network when posting or fetching, which the seed never does.
$SQL < "$seed/connectors.sql"
rpc "$admin" RefreshConnectors '{}' >/dev/null

echo "Seeding social accounts (SQL)"
$SQL < "$seed/accounts.sql"

echo "Seeding users and the Marketing group"
user_ids=()
while read -r u; do
	rpc "$admin" CreateUser "$u" >/dev/null
	name=$(jq -r .username <<<"$u")
	user_ids+=("$(rpc "$admin" GetUsers '{}' | jq --arg n "$name" '.users[] | select(.username == $n) | .id')")
done < <(jq -c '.[]' "$seed/users.json")

group_id=$(rpc "$admin" CreateUserGroup '{"name": "Marketing"}' | jq .groupId)
rpc "$admin" SetUserGroupMembers \
	"$(jq -nc --argjson g "$group_id" '{groupId: $g, userIds: $ARGS.positional | map(tonumber)}' --args "${user_ids[@]}")" >/dev/null

for account in 1 2 3 4 5; do
	rpc "$admin" SetSocialAccountShares \
		"$(jq -nc --argjson a "$account" --argjson g "$group_id" \
			'{socialAccountId: $a, shares: [{userGroupId: $g, canRead: true, canPost: true}]}')" >/dev/null
done

echo "Seeding campaigns"
while read -r c; do
	rpc "$admin" CreateCampaign "$(jq -c '{name, description}' <<<"$c")" >/dev/null
	# CreateCampaign doesn't return the new id (campaignId is always 0).
	id=$(rpc "$admin" GetCampaigns '{}' | jq --arg n "$(jq -r .name <<<"$c")" '.campaigns[] | select(.name == $n) | .id')
	for account in $(jq '.social_account_ids[]' <<<"$c"); do
		rpc "$admin" AddSocialAccountToCampaign "{\"campaignId\": $id, \"socialAccountId\": $account}" >/dev/null
	done
done < <(jq -c '.[]' "$seed/campaigns.json")

echo "Seeding canned posts"
jq -c '.[] | {content: .}' "$seed/canned-posts.json" | while read -r p; do
	rpc "$admin" CreateCannedPost "$p" >/dev/null
done

echo "Seeding publishing history and feed (SQL)"
$SQL < "$seed/history.sql"

echo "Seeding feed columns and a tagged feed post"
# Both search through the Mastodon account, answered by fake-mastodon.py.
rpc "$admin" CreateFeedColumn '{"query": "#coffee", "socialAccountIds": [1]}' >/dev/null
rpc "$admin" CreateFeedColumn '{"query": "latte art", "title": "Latte art", "socialAccountIds": [1]}' >/dev/null
rpc "$admin" GetFeed '{}' | jq -c '.posts[] | select(.remoteId == "demo-2002")
	| {post: {socialAccountId, feedPostId: .id, remoteId, remoteUrl}, tags: ["to-reply"]}' |
	while read -r t; do rpc "$admin" SetPostTags "$t" >/dev/null; done

echo "Seeding scheduled posts and drafts"
jq -c '.scheduled[]' "$seed/posts.json" | while read -r p; do
	at=$(when "$(jq -r .days <<<"$p")" "$(jq -r .time <<<"$p")")
	rpc "$admin" SubmitPost "$(jq -c --arg at "$at" \
		'{content, socialAccounts: .accounts, campaignId: .campaign, scheduledAt: $at}' <<<"$p")" >/dev/null
done
jq -c '.drafts[]' "$seed/posts.json" | while read -r p; do
	rpc "$admin" SubmitPost "$(jq -c '{content, socialAccounts: .accounts, saveAsDraft: true}' <<<"$p")" >/dev/null
done

echo "Seeding an approval policy and posts waiting for approval"
admin_id=$(rpc "$admin" GetUsers '{}' | jq '.users[] | select(.username == "admin") | .id')
rpc "$admin" CreateAccountPolicy "$(jq -nc --argjson u "$admin_id" '{
	name: "Brand review",
	description: "Posts to the main brand accounts are checked by the brand lead before they go out.",
	applyToUi: true, applyToMcp: true,
	stages: [{stageOrder: 1, userId: $u}],
	socialAccountIds: [1, 2, 3]}')" >/dev/null

login jordan "$(jq -r '.[] | select(.username == "jordan") | .password' "$seed/users.json")"
jq -c '.pending[]' "$seed/posts.json" | while read -r p; do
	at=$(when "$(jq -r .days <<<"$p")" "$(jq -r .time <<<"$p")")
	rpc "$tmp/jordan.cookies" SubmitPost "$(jq -c --arg at "$at" \
		'{content, socialAccounts: .accounts, campaignId: .campaign, scheduledAt: $at}' <<<"$p")" >/dev/null
done

$SQL < "$seed/backdate.sql"

rm -f "$tmp"/*.cookies
echo "Seed complete"
