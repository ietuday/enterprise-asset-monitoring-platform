from main import build_health_reasons, calculate_health_score, clamp_score, health_status


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
