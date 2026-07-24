# AGENTS.md

Guide for AI agents integrating with Japella.

## Project overview

Japella is a self-hosted social media posting and management tool. Backend is Go (Connect RPC); frontend is Vue.

## Authentication

MCP and the Connect API share API keys created in the Japella UI (user settings → API Keys).

For MCP, send:

```http
Authorization: Bearer <api-key>
```

Cookie sessions are not accepted on `/mcp`. Invalid or missing Bearer tokens return `401`.

## MCP endpoint

| | |
|---|---|
| URL | `{baseUrl}/mcp` |
| Transport | Streamable HTTP |
| Auth | Bearer API key (required) |

### Cursor example

```json
{
  "mcpServers": {
    "japella": {
      "url": "https://your-host/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_API_KEY"
      }
    }
  }
}
```

## MCP tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `japella_list_social_accounts` | List social accounts visible to the API key's user | `only_active` (bool, optional) — if true, only active accounts |
| `japella_submit_post` | Post (or schedule) content to one or more social accounts. May return `state: pending_approval` when an AccountPolicy applies to MCP | `content` (string, required); `social_account_ids` (number[], required); `scheduled_at` (string, optional RFC3339 or `YYYY-MM-DDTHH:MM`); `campaign_id` (number, optional) |
| `japella_list_pending_approvals` | List posts waiting for the caller's approval at the current stage | (none) |
| `japella_approve_post` | Approve the current stage; publishes/schedules when all stages pass | `post_id` (number, required) |
| `japella_reject_post` | Reject a pending post (current-stage approver or submitter) | `post_id` (number, required); `reason` (string, optional) |

### Typical flow

1. Call `japella_list_social_accounts` and note accounts where `can_post` is true.
2. Call `japella_submit_post` with `content` and those account `id` values.
3. If a response post has `state: pending_approval`, an AccountPolicy held the post. Eligible approvers use `japella_list_pending_approvals` / `japella_approve_post` (or reject).

Account access and posting permissions follow the same RBAC and social-account sharing rules as the Connect API (`GetSocialAccounts` / `SubmitPost`).

## Connect API

Agents that prefer RPC can use the Connect API under `/api` with the same Bearer API key. See protobuf definitions under `proto/japella/controlapi/v1/`.

## Discovery

- [`/llms.txt`](/llms.txt) — short index of integration endpoints
- [`/mcp`](/mcp) — MCP Streamable HTTP server

## Security notes

- Treat API keys as secrets; prefer read/write keys only where posting is required.
- `japella_submit_post` publishes to live social networks for accounts the key's user can post to, unless held for approval.
- Submitters cannot approve their own posts.
- Do not log full API keys.
