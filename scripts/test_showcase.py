#!/usr/bin/env python3
"""API acceptance: run against a fresh showcase-configured disposable server."""
import importlib.util
from pathlib import Path
import sys


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(filename))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


showcase = load('showcase', 'seed-showcase.py')
market = load('market', 'showcase-market.py')
legacy = showcase.demo
# Legacy fixed-date generation stays reproducible.
from datetime import datetime, timezone
import random
end = datetime(2026, 9, 8, tzinfo=timezone.utc)
assert legacy.build_book(random.Random(42), end) == legacy.build_book(random.Random(42), end)
assert showcase.trade_csv() == showcase.trade_csv()
assert '285A' in showcase.trade_csv()
assert market.response({'symbol': ['285A.T'], 'interval': ['D'], 'from': ['2026-09-01T00:00:00Z'],
                        'to': ['2026-09-09T00:00:00Z']})['bars'][2]['close'] == 750

# Rolling dates reuse the actual JP calendar, including holidays and year boundaries.
from datetime import date, timedelta
from showcase_dates import timeline
for today in (date(2026, 10, 8), date(2026, 1, 5), date(2024, 3, 1), date(2026, 9, 24)):
    events, sessions = timeline(today)
    assert len(events) == len(set(events)) == 6
    assert events == sorted(events) and set(events) <= set(sessions)
    assert all(today - timedelta(days=30) <= date.fromisoformat(day) < today for day in sessions)
assert '2026-09-22' not in timeline(date(2026, 9, 24))[1]
assert timeline(date(2026, 10, 8)) != timeline(date(2026, 10, 9))
for outside in (date(2020, 1, 20), date(2031, 1, 5)):
    try:
        timeline(outside)
        raise AssertionError('must refuse windows outside the versioned calendar')
    except ValueError:
        pass
# Verify scripts agree on both relocated event dates and intervening daily prices.
import os, subprocess
subprocess.run([sys.executable, '-c', """
import importlib.util, csv, io
from showcase_dates import DATES, SESSIONS
spec=importlib.util.spec_from_file_location('seed','scripts/seed-showcase.py')
seed=importlib.util.module_from_spec(spec);spec.loader.exec_module(seed)
spec=importlib.util.spec_from_file_location('market','scripts/showcase-market.py')
market=importlib.util.module_from_spec(spec);spec.loader.exec_module(market)
rows=list(csv.DictReader(io.StringIO(seed.trade_csv())))
assert {r['約定日'].replace('/','-') for r in rows} == set(DATES)
assert len(rows) == 212
assert len([code for code in market.PRICES if code.startswith('90')]) == 100
assert sum(len([r for r in rows if r['銘柄コード'] == str(9000+i)]) == 2 for i in range(100)) == 100
bars=market.response({'symbol':['285A.T'],'interval':['D'],'from':[SESSIONS[0]+'T00:00:00Z'],'to':['2026-10-08T00:00:00Z']})['bars']
assert [b['market_date'] for b in bars] == SESSIONS
assert bars[0]['close']==800 and bars[-1]['close']==750
assert market.response({'symbol':['285A.T'],'interval':['D'],'from':['2026-10-08T00:00:00Z'],'to':['2026-10-09T00:00:00Z']})['bars']==[]
"""], check=True, env={**os.environ, 'TRADELENS_SHOWCASE_TODAY':'2026-10-08', 'PYTHONPATH':str(Path(__file__).resolve().parent)})

# Existing Web E2E imports the seed module from outside scripts/.
subprocess.run([sys.executable, '-c', "import importlib.util; s=importlib.util.spec_from_file_location('showcase','../scripts/seed-showcase.py'); m=importlib.util.module_from_spec(s); s.loader.exec_module(m); assert '2026/09/01' in m.trade_csv()"],
               cwd=Path(__file__).resolve().parents[1] / 'web', check=True)

if len(sys.argv) > 1:
    api = legacy.Api(sys.argv[1])
    api('POST', '/setup', {'email': 'fictional@example.com', 'password': 'fictional-showcase-password'})
    api.login('fictional@example.com', 'fictional-showcase-password')
    showcase.seed(api)
    accounts = api('GET', '/accounts')
    account = accounts[0]['id']
    trades = api('GET', '/trades?account_id=' + account)
    assert len(trades) == 7, trades
    assert all(t['qty_remaining'] == 0 for t in trades), trades
    assert sum(t['net_pnl'] for t in trades) == 25000, trades
    assert {t['symbol'] for t in trades} == {'6501', '285A', '7203', '6758', '8306'}
    plan = next(p for p in api('GET', '/setups') if p['name'].startswith(showcase.LABEL))
    assert plan['direction'] == 'long' and plan['target_price'] == 1200, plan
    assert next(t for t in trades if t['symbol'] == '6501')['avg_exit_price'] == 1100
    values = api('GET', '/analytics/account-value?account_id=' + account + '&from=2026-09-01&to=2026-09-08')
    assert len(values['points']) == 6, values
    last = values['points'][-1]
    assert last['estimated_account_value'] == 975000, values
    assert last['contributed_capital'] == 950000, values
    assert last['realized_pnl'] == 25000, values
    assert all(p['status'] == 'complete' for p in values['points']), values
    counts = api('GET', '/news/performance')['counts']
    assert counts['total'] == 6, counts
    assert counts['pending'] == 4 and counts['unavailable'] == 2, counts
    # Importer's own dedup path, independently of seed's whole-user rerun guard.
    text = showcase.trade_csv()
    preview = showcase.upload(api, '/imports', account, text)
    result = showcase.upload(api, '/imports/commit', account, text, preview['suggested_mapping'])
    assert result['inserted'] == 0 and result['skipped'] == 14, result
    try:
        showcase.seed(api)
        raise AssertionError('rerun must refuse')
    except SystemExit:
        pass
    assert len(api('GET', '/accounts')) == 1
    assert len(api('GET', '/news')) == 3
print('showcase checks passed')
