import os
from contextlib import contextmanager

import psycopg2
from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

load_dotenv()

app = FastAPI(
    title="Report Service",
    description="Reporting and analytics service for Enterprise Asset Monitoring Platform",
    version="1.0.0",
)

DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgres://monitoring_user:monitoring_pass@localhost:5435/monitoring_db",
)


class MaintenanceInsight(BaseModel):
    asset_id: str
    asset_name: str
    health_score: int
    risk_level: str
    last_maintenance_date: str | None
    open_tasks: int
    overdue_tasks: int
    recommended_action: str
    reason: str


@contextmanager
def get_db_connection():
    connection = None
    try:
        connection = psycopg2.connect(DATABASE_URL)
        yield connection
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc))
    finally:
        if connection:
            connection.close()


@app.get("/health")
def health():
    return {
        "service": "report-service",
        "status": "healthy",
    }


@app.get("/reports/summary")
def get_summary():
    with get_db_connection() as connection:
        with connection.cursor() as cursor:
            cursor.execute("SELECT COUNT(*) FROM assets;")
            total_assets = cursor.fetchone()[0]

            cursor.execute("SELECT COUNT(*) FROM alerts;")
            total_alerts = cursor.fetchone()[0]

            cursor.execute("SELECT COUNT(*) FROM alerts WHERE status = 'OPEN';")
            open_alerts = cursor.fetchone()[0]

            cursor.execute("SELECT COUNT(*) FROM alerts WHERE status = 'ACKNOWLEDGED';")
            acknowledged_alerts = cursor.fetchone()[0]

            cursor.execute("SELECT COUNT(*) FROM alerts WHERE status = 'RESOLVED';")
            resolved_alerts = cursor.fetchone()[0]

            cursor.execute("SELECT COUNT(*) FROM alerts WHERE severity = 'CRITICAL';")
            critical_alerts = cursor.fetchone()[0]

            cursor.execute("SELECT COUNT(*) FROM alerts WHERE severity = 'HIGH';")
            high_alerts = cursor.fetchone()[0]

            return {
                "totalAssets": total_assets,
                "totalAlerts": total_alerts,
                "openAlerts": open_alerts,
                "acknowledgedAlerts": acknowledged_alerts,
                "resolvedAlerts": resolved_alerts,
                "criticalAlerts": critical_alerts,
                "highAlerts": high_alerts,
            }


@app.get("/reports/assets")
def get_asset_report():
    with get_db_connection() as connection:
        with connection.cursor() as cursor:
            cursor.execute(
                """
                SELECT status, COUNT(*)
                FROM assets
                GROUP BY status
                ORDER BY status;
                """
            )

            rows = cursor.fetchall()

            return {
                "assetsByStatus": [
                    {
                        "status": row[0],
                        "count": row[1],
                    }
                    for row in rows
                ]
            }


def table_exists(cursor, table_name):
    cursor.execute("SELECT to_regclass(%s);", (table_name,))
    return cursor.fetchone()[0] is not None


def fetch_count_map(cursor, query, params=None):
    cursor.execute(query, params or ())
    return {str(row[0]): int(row[1]) for row in cursor.fetchall()}


def health_status(score):
    if score >= 80:
        return "healthy"
    if score >= 60:
        return "warning"
    if score >= 30:
        return "degraded"
    return "critical"


def clamp_score(score):
    return max(0, min(100, score))


def build_health_reasons(
    asset_status="",
    critical_alerts=0,
    warning_alerts=0,
    open_incidents=0,
    sla_breaches=0,
    overdue_maintenance=0,
):
    reasons = []

    if str(asset_status or "").upper() in ("DOWN", "INACTIVE"):
        reasons.append("Asset status down/inactive")
    if critical_alerts:
        reasons.append(f"{critical_alerts} active critical alert")
    if warning_alerts:
        reasons.append(f"{warning_alerts} active warning alert")
    if open_incidents:
        reasons.append(f"{open_incidents} open incident")
    if sla_breaches:
        reasons.append(f"{sla_breaches} SLA breach")
    if overdue_maintenance:
        reasons.append(f"{overdue_maintenance} overdue maintenance task")

    return reasons


def calculate_health_score(
    asset_status="",
    critical_alerts=0,
    warning_alerts=0,
    open_incidents=0,
    sla_breaches=0,
    overdue_maintenance=0,
):
    score = 100

    if str(asset_status or "").upper() in ("DOWN", "INACTIVE"):
        score -= 50
    score -= critical_alerts * 25
    score -= warning_alerts * 10
    score -= open_incidents * 20
    score -= sla_breaches * 20
    score -= overdue_maintenance * 15

    return clamp_score(score)


def calculate_risk_level(health_score: int | float, overdue_tasks: int) -> str:
    if health_score >= 80:
        base_level = "low"
    elif health_score >= 60:
        base_level = "medium"
    elif health_score >= 40:
        base_level = "high"
    else:
        base_level = "critical"

    levels = ["low", "medium", "high", "critical"]
    risk_index = levels.index(base_level)
    if overdue_tasks > 0:
        risk_index = min(risk_index + 1, len(levels) - 1)

    return levels[risk_index]


def build_recommendation(
    risk_level: str,
    health_score: int | float,
    overdue_tasks: int,
) -> tuple[str, str]:
    overdue_text = f"{overdue_tasks} overdue maintenance task"
    if overdue_tasks != 1:
        overdue_text += "s"

    if risk_level == "low":
        return (
            "No immediate action required",
            "Asset health is healthy and there are no urgent maintenance issues",
        )

    if risk_level == "medium":
        if overdue_tasks > 0:
            reason = f"Risk was escalated by {overdue_text}"
        else:
            reason = f"Asset health score is moderate at {health_score}"
        return "Monitor asset and review upcoming maintenance", reason

    if risk_level == "high":
        reasons = []
        if health_score < 60:
            reasons.append(f"Asset health score is low at {health_score}")
        if overdue_tasks > 0:
            reasons.append(f"there are {overdue_text}")
        return (
            "Schedule preventive maintenance within 7 days",
            " and ".join(reasons) if reasons else "Asset requires preventive maintenance review",
        )

    reasons = []
    if health_score < 40:
        reasons.append(f"Asset health score is critical at {health_score}")
    elif health_score < 60:
        reasons.append(f"Asset health score is low at {health_score}")
    if overdue_tasks > 0:
        reasons.append(f"there are {overdue_text}")
    return (
        "Immediate maintenance attention required",
        " and ".join(reasons) if reasons else "Asset requires immediate maintenance attention",
    )


@app.get("/reports/asset-health")
def get_asset_health():
    return build_asset_health()


@app.get("/reports/asset-health/{asset_id}")
def get_one_asset_health(asset_id: str):
    rows = build_asset_health(asset_id)
    if not rows:
        raise HTTPException(status_code=404, detail="asset not found")
    return rows[0]


@app.get("/reports/maintenance-insights", response_model=list[MaintenanceInsight])
def get_maintenance_insights():
    return build_maintenance_insights()


def build_maintenance_insights():
    health_rows = build_asset_health()
    if not health_rows:
        return []

    open_tasks = {}
    overdue_tasks = {}
    last_maintenance_dates = {}

    with get_db_connection() as connection:
        with connection.cursor() as cursor:
            if table_exists(cursor, "maintenance_tasks"):
                open_tasks = fetch_count_map(
                    cursor,
                    """
                    SELECT asset_id, COUNT(*)
                    FROM maintenance_tasks
                    WHERE status NOT IN ('completed', 'cancelled')
                    GROUP BY asset_id;
                    """,
                )
                overdue_tasks = fetch_count_map(
                    cursor,
                    """
                    SELECT asset_id, COUNT(*)
                    FROM maintenance_tasks
                    WHERE due_date < NOW()
                    AND status NOT IN ('completed', 'cancelled')
                    GROUP BY asset_id;
                    """,
                )
                cursor.execute(
                    """
                    SELECT asset_id, MAX(completed_at)::date
                    FROM maintenance_tasks
                    WHERE completed_at IS NOT NULL
                    GROUP BY asset_id;
                    """
                )
                last_maintenance_dates = {
                    str(row[0]): row[1].isoformat() if row[1] else None
                    for row in cursor.fetchall()
                }

    insights = []
    for row in health_rows:
        asset_id = str(row["asset_id"])
        health_score = int(row["health_score"])
        overdue_count = overdue_tasks.get(asset_id, 0)
        risk_level = calculate_risk_level(health_score, overdue_count)
        recommended_action, reason = build_recommendation(
            risk_level,
            health_score,
            overdue_count,
        )

        insights.append(
            {
                "asset_id": asset_id,
                "asset_name": row.get("asset_name") or asset_id,
                "health_score": health_score,
                "risk_level": risk_level,
                "last_maintenance_date": last_maintenance_dates.get(asset_id),
                "open_tasks": open_tasks.get(asset_id, 0),
                "overdue_tasks": overdue_count,
                "recommended_action": recommended_action,
                "reason": reason,
            }
        )

    return insights


def build_asset_health(asset_id=None):
    with get_db_connection() as connection:
        with connection.cursor() as cursor:
            params = (asset_id,) if asset_id else ()
            where_clause = "WHERE id = %s" if asset_id else ""
            cursor.execute(
                f"""
                SELECT id, name, status
                FROM assets
                {where_clause}
                ORDER BY name;
                """,
                params,
            )
            assets = cursor.fetchall()

            asset_ids = [str(row[0]) for row in assets]
            if not asset_ids:
                return []

            critical_alerts = {}
            warning_alerts = {}
            open_incidents = {}
            sla_breaches = {}
            overdue_maintenance = {}

            if table_exists(cursor, "alerts"):
                critical_alerts = fetch_count_map(
                    cursor,
                    """
                    SELECT asset_id, COUNT(*)
                    FROM alerts
                    WHERE status IN ('OPEN', 'ACKNOWLEDGED')
                    AND severity = 'CRITICAL'
                    GROUP BY asset_id;
                    """,
                )
                warning_alerts = fetch_count_map(
                    cursor,
                    """
                    SELECT asset_id, COUNT(*)
                    FROM alerts
                    WHERE status IN ('OPEN', 'ACKNOWLEDGED')
                    AND severity IN ('HIGH', 'MEDIUM', 'WARNING')
                    GROUP BY asset_id;
                    """,
                )

            if table_exists(cursor, "incidents"):
                open_incidents = fetch_count_map(
                    cursor,
                    """
                    SELECT asset_id, COUNT(*)
                    FROM incidents
                    WHERE status IN ('OPEN', 'ASSIGNED', 'ACKNOWLEDGED')
                    GROUP BY asset_id;
                    """,
                )

            if table_exists(cursor, "incident_sla_tracking") and table_exists(cursor, "incidents"):
                sla_breaches = fetch_count_map(
                    cursor,
                    """
                    SELECT incidents.asset_id, COUNT(*)
                    FROM incident_sla_tracking
                    JOIN incidents ON incidents.id = incident_sla_tracking.incident_id
                    WHERE incident_sla_tracking.status IN ('ACK_BREACHED', 'RESOLUTION_BREACHED', 'ESCALATED')
                    GROUP BY incidents.asset_id;
                    """,
                )

            if table_exists(cursor, "maintenance_tasks"):
                overdue_maintenance = fetch_count_map(
                    cursor,
                    """
                    SELECT asset_id, COUNT(*)
                    FROM maintenance_tasks
                    WHERE due_date < NOW()
                    AND status NOT IN ('completed', 'cancelled')
                    GROUP BY asset_id;
                    """,
                )

            results = []
            for asset in assets:
                current_asset_id = str(asset[0])
                asset_name = asset[1]
                asset_status = str(asset[2] or "")

                critical_count = critical_alerts.get(current_asset_id, 0)
                warning_count = warning_alerts.get(current_asset_id, 0)
                incident_count = open_incidents.get(current_asset_id, 0)
                breach_count = sla_breaches.get(current_asset_id, 0)
                maintenance_count = overdue_maintenance.get(current_asset_id, 0)
                score = calculate_health_score(
                    asset_status=asset_status,
                    critical_alerts=critical_count,
                    warning_alerts=warning_count,
                    open_incidents=incident_count,
                    sla_breaches=breach_count,
                    overdue_maintenance=maintenance_count,
                )
                reasons = build_health_reasons(
                    asset_status=asset_status,
                    critical_alerts=critical_count,
                    warning_alerts=warning_count,
                    open_incidents=incident_count,
                    sla_breaches=breach_count,
                    overdue_maintenance=maintenance_count,
                )
                results.append(
                    {
                        "asset_id": current_asset_id,
                        "asset_name": asset_name,
                        "health_score": score,
                        "health_status": health_status(score),
                        "reasons": reasons,
                    }
                )

            return results


@app.get("/reports/alerts")
def get_alert_report():
    with get_db_connection() as connection:
        with connection.cursor() as cursor:
            cursor.execute(
                """
                SELECT status, COUNT(*)
                FROM alerts
                GROUP BY status
                ORDER BY status;
                """
            )
            status_rows = cursor.fetchall()

            cursor.execute(
                """
                SELECT severity, COUNT(*)
                FROM alerts
                GROUP BY severity
                ORDER BY severity;
                """
            )
            severity_rows = cursor.fetchall()

            cursor.execute(
                """
                SELECT asset_id, COUNT(*)
                FROM alerts
                GROUP BY asset_id
                ORDER BY COUNT(*) DESC;
                """
            )
            asset_rows = cursor.fetchall()

            return {
                "alertsByStatus": [
                    {
                        "status": row[0],
                        "count": row[1],
                    }
                    for row in status_rows
                ],
                "alertsBySeverity": [
                    {
                        "severity": row[0],
                        "count": row[1],
                    }
                    for row in severity_rows
                ],
                "alertsByAsset": [
                    {
                        "assetId": row[0],
                        "count": row[1],
                    }
                    for row in asset_rows
                ],
            }
