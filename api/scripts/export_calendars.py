"""Export audited, bounded session snapshots; run with exchange-calendars==4.13.2."""
from pathlib import Path
import exchange_calendars as xc

assert xc.__version__ == "4.13.2"
out = Path(__file__).resolve().parents[1] / "internal/marketdata/calendars"
for market, name in (("JP", "XTKS"), ("US", "XNYS")):
    calendar = xc.get_calendar(name, start="2020-01-01", end="2030-12-31")
    lines = ["date,open,close"]
    for day, row in calendar.schedule.iterrows():
        lines.append(f"{day.date()},{row['open'].isoformat()},{row['close'].isoformat()}")
    (out / f"{market}.csv").write_text("\n".join(lines) + "\n")
