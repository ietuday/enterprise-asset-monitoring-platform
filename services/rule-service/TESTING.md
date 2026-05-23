# Rule Service Testing

## Rule Lifecycle Management

### Create draft rule

```bash
curl -X POST http://localhost:5004/rules \
  -H "Content-Type: application/json" \
  -H "X-User-Email: admin@example.com" \
  -d '{
    "name": "High CPU Usage",
    "metric": "cpu",
    "operator": ">",
    "threshold": 80,
    "severity": "critical"
  }'

  Expected:

{
  "status": "draft",
  "enabled": false
}
Activate rule
curl -X PATCH http://localhost:5004/rules/1/activate \
  -H "X-User-Email: admin@example.com"

Expected:

{
  "status": "active",
  "enabled": true
}
Disable rule
curl -X PATCH http://localhost:5004/rules/1/disable \
  -H "X-User-Email: admin@example.com"

Expected:

{
  "status": "disabled",
  "enabled": false
}
Archive rule
curl -X PATCH http://localhost:5004/rules/1/archive \
  -H "X-User-Email: admin@example.com"

Expected:

{
  "status": "archived",
  "enabled": false
}
Rule Versioning and Rollback
Update rule to create version snapshot
curl -X PUT http://localhost:5004/rules/1 \
  -H "Content-Type: application/json" \
  -H "X-User-Email: admin@example.com" \
  -d '{
    "name": "High CPU Usage Updated",
    "metric": "cpu",
    "operator": ">",
    "threshold": 90,
    "severity": "critical",
    "status": "draft"
  }'

Expected:

Previous rule state is stored in rule_versions
Audit history contains RULE_UPDATED
List rule versions
curl http://localhost:5004/rules/1/versions

Expected:

[
  {
    "rule_id": 1,
    "version": 1,
    "name": "High CPU Usage",
    "threshold": 80
  }
]
Rollback rule
curl -X POST http://localhost:5004/rules/1/rollback/1 \
  -H "X-User-Email: admin@example.com"

Expected:

Rule is restored to version 1 values
Audit history contains RULE_ROLLED_BACK
Prometheus rules are regenerated
Check audit history
curl http://localhost:5004/rules/1/history

Expected audit events:

RULE_CREATED
RULE_UPDATED
RULE_ROLLED_BACK