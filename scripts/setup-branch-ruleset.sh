#!/usr/bin/env bash

set -euo pipefail

OWNER="${OWNER:-ietuday}"
REPO="${REPO:-enterprise-asset-monitoring-platform}"
BRANCH_PATTERN="${BRANCH_PATTERN:-master}"
RULESET_NAME="${RULESET_NAME:-Protect master}"
ENFORCEMENT="${ENFORCEMENT:-active}"

echo "Setting up GitHub branch ruleset"
echo "Repository: $OWNER/$REPO"
echo "Branch pattern: $BRANCH_PATTERN"
echo "Ruleset name: $RULESET_NAME"
echo "Enforcement: $ENFORCEMENT"
echo ""

if ! command -v gh >/dev/null 2>&1; then
  echo "GitHub CLI 'gh' is not installed."
  echo "Install it from: https://cli.github.com/"
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "You are not logged in to GitHub CLI."
  echo "Run: gh auth login"
  exit 1
fi

REPO_ID=$(gh api "repos/$OWNER/$REPO" --jq '.id')

echo "Repository ID: $REPO_ID"
echo ""

EXISTING_RULESET_ID=$(gh api "repos/$OWNER/$REPO/rulesets" \
  --jq ".[] | select(.name == \"$RULESET_NAME\") | .id" || true)

if [[ -n "$EXISTING_RULESET_ID" ]]; then
  echo "Ruleset already exists: $RULESET_NAME"
  echo "Ruleset ID: $EXISTING_RULESET_ID"
  echo ""
  read -r -p "Do you want to delete and recreate it? [y/N]: " confirm

  if [[ "$confirm" =~ ^[Yy]$ ]]; then
    gh api \
      --method DELETE \
      "repos/$OWNER/$REPO/rulesets/$EXISTING_RULESET_ID"

    echo "Deleted existing ruleset."
  else
    echo "Cancelled."
    exit 0
  fi
fi

cat > /tmp/branch-ruleset.json <<EOF
{
  "name": "$RULESET_NAME",
  "target": "branch",
  "enforcement": "$ENFORCEMENT",
  "conditions": {
    "ref_name": {
      "include": [
        "refs/heads/$BRANCH_PATTERN"
      ],
      "exclude": []
    }
  },
  "bypass_actors": [
    {
      "actor_id": 5,
      "actor_type": "RepositoryRole",
      "bypass_mode": "always"
    }
  ],
  "rules": [
    {
      "type": "deletion"
    },
    {
      "type": "non_fast_forward"
    },
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 1,
        "dismiss_stale_reviews_on_push": true,
        "require_code_owner_review": false,
        "require_last_push_approval": false,
        "required_review_thread_resolution": true,
        "automatic_copilot_code_review_enabled": true,
        "allowed_merge_methods": [
          "merge",
          "squash",
          "rebase"
        ]
      }
    },
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": true,
        "do_not_enforce_on_create": true,
        "required_status_checks": [
          {
            "context": "Validate Docker Compose"
          },
          {
            "context": "Script Checks"
          },
          {
            "context": "Go Services (asset-service)"
          },
          {
            "context": "Go Services (telemetry-service)"
          },
          {
            "context": "Go Services (alert-service)"
          },
          {
            "context": "Go Services (rule-service)"
          },
          {
            "context": "Node Services (api-gateway)"
          },
          {
            "context": "Node Services (auth-service)"
          },
          {
            "context": "Python Report Service"
          },
          {
            "context": "React Dashboard"
          },
          {
            "context": "Docker Compose Build"
          }
        ]
      }
    }
  ]
}
EOF

echo "Creating ruleset..."

gh api \
  --method POST \
  "repos/$OWNER/$REPO/rulesets" \
  --input /tmp/branch-ruleset.json \
  --jq '.id'

echo ""
echo "Ruleset created successfully."
echo ""
echo "Open:"
echo "https://github.com/$OWNER/$REPO/settings/rules"