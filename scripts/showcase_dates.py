"""Shared fictional showcase timeline; fixed unless the demo runtime opts in."""
import csv
from datetime import date, timedelta
import os
from pathlib import Path

FIXED_DATES = ['2026-09-01', '2026-09-02', '2026-09-03', '2026-09-04', '2026-09-07', '2026-09-08']


def timeline(today):
    calendar = Path(__file__).resolve().parents[1] / 'api/internal/marketdata/calendars/JP.csv'
    with calendar.open() as source:
        available = [row['date'] for row in csv.DictReader(source)]
    start = (today - timedelta(days=30)).isoformat()
    # Fail closed outside the versioned calendar rather than inventing sessions.
    if not available[0] <= start <= today.isoformat() <= available[-1]:
        raise ValueError('Demo startup date is outside the JP calendar coverage')
    sessions = [day for day in available if start <= day < today.isoformat()]
    if len(sessions) < 6:
        raise ValueError('Insufficient JP sessions for the fictional showcase')
    return [sessions[i * (len(sessions) - 1) // 5] for i in range(6)], sessions


startup_day = os.environ.get('TRADELENS_SHOWCASE_TODAY')
DATES, SESSIONS = timeline(date.fromisoformat(startup_day)) if startup_day else (FIXED_DATES, FIXED_DATES)
DATE_MAP = dict(zip(FIXED_DATES, DATES))
