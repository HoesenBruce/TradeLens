"""Synthetic Generic Bars v1 provider for disposable issue #307 browser QA."""
from datetime import datetime, timedelta, timezone
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
from urllib.parse import parse_qs, urlparse


class Fixture(BaseHTTPRequestHandler):
    def do_GET(self):
        query = parse_qs(urlparse(self.path).query)
        start = datetime.fromisoformat(query['from'][0].replace('Z', '+00:00'))
        end = datetime.fromisoformat(query['to'][0].replace('Z', '+00:00'))
        interval = query['interval'][0]
        step = timedelta(days=1) if interval == 'D' else timedelta(minutes=int(interval))
        bars = []
        stamp = start.replace(hour=0, minute=0, second=0, microsecond=0) if interval == 'D' else start
        while stamp <= start:
            stamp += step
        while stamp < end and len(bars) < 10000:
            price = 105 + len(bars) % 5
            bars.append({'timestamp': stamp.isoformat(), 'market_date': stamp.astimezone(timezone(timedelta(hours=9))).date().isoformat(), 'open': price, 'high': price+4, 'low': price-4, 'close': price+1, 'volume': 1000})
            stamp += step
        assert all(b['low'] <= b['open'] <= b['high'] for b in bars)
        payload = json.dumps({'symbol': query['symbol'][0], 'interval': interval, 'source': 'synthetic-issue307', 'timezone': 'Asia/Tokyo', 'adjustment_status': 'unadjusted', 'corporate_actions': [], 'bars': bars}).encode()
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)


if __name__ == '__main__':
    HTTPServer(('127.0.0.1', 8093), Fixture).serve_forever()
