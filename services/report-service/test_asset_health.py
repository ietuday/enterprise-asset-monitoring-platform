import pytest
from fastapi import HTTPException

import main
from main import (
    build_asset_health,
    build_health_reasons,
    calculate_health_score,
    clamp_score,
    fetch_count_map,
    get_alert_report,
    get_asset_report,
    get_one_asset_health,
    get_summary,
    health,
    health_status,
    table_exists,
)


def test_health_status_boundaries():
    assert health_status(100) == "healthy"
    assert health_status(75) == "warning"
    assert health_status(45) == "degraded"
    assert health_status(10) == "critical"


def test_score_is_clamped():
    assert clamp_score(120) == 100
    assert clamp_score(-20) == 0
    assert calculate_health_score(critical_alerts=10) == 0


def test_health_score_penalties():
    assert calculate_health_score(asset_status="DOWN") == 50
    assert calculate_health_score(critical_alerts=1) == 75
    assert calculate_health_score(warning_alerts=1) == 90
    assert calculate_health_score(open_incidents=1) == 80
    assert calculate_health_score(sla_breaches=1) == 80
    assert calculate_health_score(overdue_maintenance=1) == 85


def test_combined_health_score_and_reasons():
    score = calculate_health_score(
        asset_status="INACTIVE",
        critical_alerts=1,
        warning_alerts=2,
        open_incidents=1,
        sla_breaches=1,
        overdue_maintenance=1,
    )
    reasons = build_health_reasons(
        asset_status="INACTIVE",
        critical_alerts=1,
        warning_alerts=2,
        open_incidents=1,
        sla_breaches=1,
        overdue_maintenance=1,
    )

    assert score == 0
    assert reasons == [
        "Asset status down/inactive",
        "1 active critical alert",
        "2 active warning alert",
        "1 open incident",
        "1 SLA breach",
        "1 overdue maintenance task",
    ]


def test_health_helpers_return_expected_empty_and_status_values():
    assert health() == {"service": "report-service", "status": "healthy"}
    assert calculate_health_score() == 100
    assert build_health_reasons() == []
    assert build_health_reasons(asset_status="down") == ["Asset status down/inactive"]
    assert build_health_reasons(
        critical_alerts=2,
        warning_alerts=1,
        open_incidents=3,
        sla_breaches=1,
        overdue_maintenance=2,
    ) == [
        "2 active critical alert",
        "1 active warning alert",
        "3 open incident",
        "1 SLA breach",
        "2 overdue maintenance task",
    ]


def test_summary_assets_and_alert_reports(monkeypatch):
    cursor = FakeCursor(
        fetchone_values=[(2,), (5,), (3,), (1,), (1,), (2,), (1,)],
        fetchall_values=[
            [("ACTIVE", 2), ("DOWN", 1)],
            [("OPEN", 3), ("RESOLVED", 2)],
            [("CRITICAL", 2), ("HIGH", 1)],
            [("asset-1", 4), ("asset-2", 1)],
        ],
    )
    monkeypatch.setattr(main, "get_db_connection", lambda: FakeConnectionContext(cursor))

    assert get_summary() == {
        "totalAssets": 2,
        "totalAlerts": 5,
        "openAlerts": 3,
        "acknowledgedAlerts": 1,
        "resolvedAlerts": 1,
        "criticalAlerts": 2,
        "highAlerts": 1,
    }
    assert get_asset_report()["assetsByStatus"] == [
        {"status": "ACTIVE", "count": 2},
        {"status": "DOWN", "count": 1},
    ]
    assert get_alert_report()["alertsByAsset"] == [
        {"assetId": "asset-1", "count": 4},
        {"assetId": "asset-2", "count": 1},
    ]


def test_table_exists_and_fetch_count_map():
    cursor = FakeCursor(
        fetchone_values=[("public.alerts",), (None,)],
        fetchall_values=[[("asset-1", 2), ("asset-2", "3")]],
    )

    assert table_exists(cursor, "alerts") is True
    assert table_exists(cursor, "missing_table") is False
    assert fetch_count_map(cursor, "SELECT asset_id, COUNT(*) FROM alerts") == {
        "asset-1": 2,
        "asset-2": 3,
    }


def test_asset_health_builds_scores_with_all_optional_tables(monkeypatch):
    cursor = FakeCursor(
        fetchone_values=[
            ("alerts",),
            ("incidents",),
            ("incident_sla_tracking",),
            ("incidents",),
            ("maintenance_tasks",),
        ],
        fetchall_values=[
            [(1, "Boiler 1", "ACTIVE"), (2, "Pump 2", "DOWN")],
            [(1, 1)],
            [(1, 2)],
            [(1, 1), (2, 1)],
            [(2, 1)],
            [(1, 1), (2, 1)],
        ],
    )
    monkeypatch.setattr(main, "get_db_connection", lambda: FakeConnectionContext(cursor))

    result = build_asset_health()

    assert result[0] == {
        "asset_id": "1",
        "asset_name": "Boiler 1",
        "health_score": 20,
        "health_status": "critical",
        "reasons": [
            "1 active critical alert",
            "2 active warning alert",
            "1 open incident",
            "1 overdue maintenance task",
        ],
    }
    assert result[1]["health_score"] == 0
    assert "Asset status down/inactive" in result[1]["reasons"]
    assert cursor.executed[0][1] == ()


def test_one_asset_health_and_not_found(monkeypatch):
    monkeypatch.setattr(
        main,
        "build_asset_health",
        lambda asset_id=None: [{"asset_id": asset_id, "asset_name": "Pump", "health_score": 100}]
        if asset_id == "1"
        else [],
    )

    assert get_one_asset_health("1")["asset_id"] == "1"
    with pytest.raises(HTTPException) as exc:
        get_one_asset_health("404")
    assert exc.value.status_code == 404


def test_asset_health_returns_empty_when_asset_query_has_no_rows(monkeypatch):
    cursor = FakeCursor(fetchall_values=[[]])
    monkeypatch.setattr(main, "get_db_connection", lambda: FakeConnectionContext(cursor))

    assert build_asset_health("missing") == []
    assert cursor.executed[0][1] == ("missing",)


class FakeConnectionContext:
    def __init__(self, cursor):
        self.connection = FakeConnection(cursor)

    def __enter__(self):
        return self.connection

    def __exit__(self, exc_type, exc, traceback):
        return False


class FakeConnection:
    def __init__(self, cursor):
        self._cursor = cursor

    def cursor(self):
        return self._cursor


class FakeCursor:
    def __init__(self, fetchone_values=None, fetchall_values=None):
        self.fetchone_values = list(fetchone_values or [])
        self.fetchall_values = list(fetchall_values or [])
        self.executed = []

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, traceback):
        return False

    def execute(self, query, params=None):
        self.executed.append((query, params))

    def fetchone(self):
        return self.fetchone_values.pop(0)

    def fetchall(self):
        return self.fetchall_values.pop(0)
