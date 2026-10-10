#!/usr/bin/env python3
"""Loopback-only fictional Generic Bars fixture. Never use with a real portfolio."""
import argparse
import json
from datetime import datetime, timezone
import sys
from pathlib import Path

# Also support existing importlib callers outside the scripts directory.
sys.path.insert(0, str(Path(__file__).resolve().parent))
from showcase_dates import DATES, SESSIONS, startup_day
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import parse_qs, urlparse

FETCHED_AT = datetime.now(timezone.utc).isoformat() if startup_day else '2026-09-09T00:00:00Z'
PRICES = {'1306': [100] * 6, '6501': [1000, 1100, 1100, 1100, 1100, 1100],
          '285A': [800, 780, 750, 750, 750, 750], '7203': [2000, 2000, 1950, 1900, 1900, 1900],
          '6758': [1500, 1500, 1500, 1500, 1550, 1550], '8306': [900, 900, 900, 950, 950, 950]}

if startup_day:
    PRICES.update({str(9000 + i): [1000] * 6 for i in range(100)})


def response(query):
    symbol = query['symbol'][0]
    code = symbol.removesuffix('.T')
    if code not in PRICES or query['interval'][0] != 'D':
        return None
    start = datetime.fromisoformat(query['from'][0].replace('Z', '+00:00'))
    end = datetime.fromisoformat(query['to'][0].replace('Z', '+00:00'))
    bars = []
    for day in SESSIONS:
        # Piecewise fictional prices also cover sessions between showcase events.
        price = PRICES[code][max(i for i, event in enumerate(DATES) if event <= day)]
        stamp = day + 'T06:00:00Z'
        if start <= datetime.fromisoformat(stamp.replace('Z', '+00:00')) < end:
            bars.append({'timestamp': stamp, 'market_date': day, 'open': price, 'high': price,
                         'low': price, 'close': price, 'volume': 100, 'fetched_at': FETCHED_AT})
    return {'symbol': symbol, 'interval': 'D', 'source': 'FICTIONAL-showcase-v1',
            'timezone': 'Asia/Tokyo', 'adjustment_status': 'unadjusted', 'bars': bars}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        parsed = urlparse(self.path)
        try:
            body = response(parse_qs(parsed.query)) if parsed.path == '/v1/bars' else None
        except (KeyError, IndexError, ValueError, TypeError):
            body = None
        self.send_response(200 if body is not None else 404)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(body or {'error': 'Outside fictional fixture coverage'}).encode())


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--port', type=int, default=18964)
    args = parser.parse_args()
    print(f'FICTIONAL prices only; coverage {SESSIONS[0]} through {SESSIONS[-1]}', flush=True)
    HTTPServer(('127.0.0.1', args.port), Handler).serve_forever()
