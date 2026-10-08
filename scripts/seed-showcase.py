#!/usr/bin/env python3
"""Fictional TradeLens showcase. Requires an empty, disposable user account."""
import csv
import importlib.util
import io
import json
import urllib.request

spec = importlib.util.spec_from_file_location('demo', __file__.replace('seed-showcase.py', 'seed-demo.py'))
demo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(demo)

LABEL = '[FICTIONAL showcase v1]'
# Fixed dates, quantities and prices: no market downloads or random state.
ROWS = [
    ('2026/09/01', '6501', '現物買', 100, 1000, 0),
    ('2026/09/02', '6501', '現物売', 100, 1100, 0),
    ('2026/09/01', '285A', '信用新規買', 100, 800, 0),
    ('2026/09/03', '285A', '信用返済売', 100, 750, 0),
    ('2026/09/02', '7203', '信用新規売', 100, 2000, 0),
    ('2026/09/04', '7203', '信用返済買', 100, 1900, 0),
    ('2026/09/03', '6758', '信用新規買', 100, 1500, 0),
    ('2026/09/04', '6758', '現引', 100, 1500, 0),
    ('2026/09/07', '6758', '現物売', 100, 1550, 0),
    ('2026/09/04', '8306', '現物買', 100, 900, 0),
    ('2026/09/04', '8306', '信用新規売', 100, 950, 0),
    ('2026/09/08', '8306', '現渡', 100, 950, 0),
]


def upload(api, path, account, text, mapping=None):
    boundary = 'tradelens-fictional-showcase-v1'
    fields = {'account_id': account, 'source_tz': 'Asia/Tokyo'}
    if mapping is not None:
        fields['column_mapping'] = json.dumps(mapping)
    parts = []
    for key, value in fields.items():
        parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{value}\r\n')
    parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="fictional-showcase.csv"\r\nContent-Type: text/csv\r\n\r\n{text}\r\n--{boundary}--\r\n')
    req = urllib.request.Request(api.base + path, data=''.join(parts).encode(), method='POST', headers={
        'Authorization': 'Bearer ' + api.token, 'Content-Type': 'multipart/form-data; boundary=' + boundary})
    with urllib.request.urlopen(req, timeout=api.timeout) as response:
        return json.load(response)


def trade_csv():
    out = io.StringIO()
    writer = csv.writer(out)
    writer.writerow(['約定日', '銘柄', '銘柄コード', '市場', '取引', '約定数量', '約定単価', '手数料/諸経費等'])
    for day, symbol, action, qty, price, fee in ROWS:
        writer.writerow([day, LABEL, symbol, '東証', action, qty, price, fee])
    return out.getvalue()


def seed(api):
    # Refuse reruns before any mutation. Partial failures also require recreation.
    if api('GET', '/accounts') or api('GET', '/news'):
        raise SystemExit('Showcase requires an empty disposable user. Recreate the database/user; no automatic deletion.')
    account = api('POST', '/accounts', {'name': LABEL + ' SBI JPY', 'broker': 'SBI Securities',
        'account_type': 'margin', 'base_currency': 'JPY', 'starting_balance': 0})['id']
    for kind, amount, day in [('deposit', 1000000, '01'), ('withdrawal', -50000, '08')]:
        api('POST', '/cash-transactions', {'account_id': account, 'type': kind, 'amount': amount,
            'currency': 'JPY', 'note': LABEL, 'occurred_at': f'2026-09-{day}T00:00:00Z'})
    text = trade_csv()
    preview = upload(api, '/imports', account, text)
    if preview['detected_broker'] != 'SBI Securities (Execution History)':
        raise SystemExit('SBI detection failed')
    result = upload(api, '/imports/commit', account, text, preview['suggested_mapping'])
    if result.get('errors') or result['inserted'] != 14:
        raise SystemExit(f'Unexpected import result: {result}')
    trades = api('GET', '/trades?account_id=' + account)
    plan = api('POST', '/setups', {'name': LABEL + ' Long plan', 'thesis': LABEL,
        'symbol': '6501', 'direction': 'long', 'target_price': 1200})
    for trade in trades:
        patch = {'notes': LABEL + ' Synthetic execution journal'}
        if trade['symbol'] == '6501':
            patch.update(target_price=1200, setup_ids=[plan['id']])
        api('PATCH', '/trades/' + trade['id'], patch)
    for symbol, market, direction, evaluate in [('6501', 'JP', 'bullish', False),
                                               ('285A', 'JP', 'bearish', True),
                                               ('7203', 'UNKNOWN', 'neutral', True)]:
        news = api('POST', '/news', {'title': LABEL + ' Synthetic catalyst ' + symbol,
            'source': LABEL, 'published_at': '2026-09-01T00:00:00Z',
            'original_text': 'Fictional exercise; no real company announcement or investment claim.',
            'summary': LABEL + ' Practice thesis', 'category': 'fictional', 'notes': LABEL,
            'assets': [{'asset_type': 'stock', 'symbol': symbol, 'market': market,
                        'exchange': 'TSE', 'display_name': LABEL + ' ' + symbol}]})
        prediction = api('POST', '/news/' + news['id'] + '/predictions', {
            'news_asset_id': news['assets'][0]['id'], 'direction': direction, 'confidence': 60,
            'reasoning': LABEL, 'catalysts': LABEL, 'risks': 'Entirely synthetic',
            'invalidation': 'Practice only', 'horizons': [1, 5]})
        if evaluate:
            api('POST', '/news/' + news['id'] + '/predictions/' + prediction['id'] + '/validate', {})
    print(json.dumps({'fictional': True, 'account_id': account, 'import': result,
        'trades': len(trades), 'news': 3, 'predictions': 3,
        'performance': api('GET', '/news/performance')['counts']}, ensure_ascii=False, indent=2))
