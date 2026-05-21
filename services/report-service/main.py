import os
from contextlib import contextmanager

import psycopg2
from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException

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