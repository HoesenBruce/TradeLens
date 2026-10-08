#!/usr/bin/env python3
"""Disposable demo supervisor. No public listener exists until seeding succeeds."""
import importlib.util
from datetime import datetime, timedelta, timezone
import json
import os
from pathlib import Path
import secrets
import signal
import sqlite3
import subprocess
import sys
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler, HTTPServer
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = ROOT / 'scripts'
EMAIL = 'demo@example.com'
PASSWORD = 'fictional-demo-password'
API = 'http://127.0.0.1:18080/api/v1'
# All children use the same Tokyo startup date, including across midnight.
os.environ['TRADELENS_SHOWCASE_TODAY'] = datetime.now(timezone(timedelta(hours=9))).date().isoformat()
sys.path.insert(0, str(SCRIPTS))
from showcase_dates import SESSIONS
MARKET_PROBE = ('http://127.0.0.1:18964/v1/bars?symbol=285A.T&interval=D'
                f'&from={SESSIONS[0]}T00:00:00Z&to={os.environ["TRADELENS_SHOWCASE_TODAY"]}T00:00:00Z')


def get(url):
    with urllib.request.urlopen(url, timeout=2) as response:
        return json.load(response)


def wait_api(process):
    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError('API exited during startup')
        try:
            get('http://127.0.0.1:18080/healthz')
            return
        except (OSError, ValueError):
            time.sleep(0.2)
    raise RuntimeError('API did not become healthy')


def stop(process):
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=15)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()


def api_env(directory, jwt, demo):
    # Do not carry accidental production DB URLs, keys or integrations into bootstrap.
    env = {k: v for k, v in os.environ.items() if not k.startswith('TM_')}
    env.update(TM_HTTP_HOST='127.0.0.1', TM_HTTP_PORT='18080',
               TM_DATABASE_URL='sqlite://' + str(directory / 'demo.db'),
               TM_ATTACH_DIR=str(directory / 'attachments'), TM_JWT_SECRET=jwt,
               TM_DEMO_MODE=str(demo).lower(), TM_ALLOW_REGISTRATION='false',
               TM_ALLOW_INSECURE_JWT='false', TM_DEFAULT_CURRENCY='JPY',
               TM_MARKET_DATA_PROVIDER='http', TM_MARKET_DATA_ENABLED='true',
               TM_MARKET_DATA_HTTP_BASE_URL='http://127.0.0.1:18964',
               TM_ECON_CALENDAR_ENABLED='false', TM_JOBS_ENABLED='false',
               TM_OCR_ENABLED='false', TM_COACH_ENABLED='false', TM_SHARE_LINKS_ENABLED='false')
    return env


def seed():
    spec = importlib.util.spec_from_file_location('demo_seed', SCRIPTS / 'seed-demo.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    api = module.Api(API)
    api('POST', '/setup', {'email': EMAIL, 'password': PASSWORD})
    # Use the existing CLI and its showcase-mode dispatch as the source of truth.
    subprocess.run([sys.executable, str(SCRIPTS / 'seed-demo.py'), '--mode', 'showcase',
                    '--api', API, '--email', EMAIL, '--password', PASSWORD], check=True)
    api.login(EMAIL, PASSWORD)
    api('PATCH', '/me/preferences', {'displayCurrency': 'JPY', 'timezone': 'Asia/Tokyo',
        'marketTimezone': 'Asia/Tokyo', 'timeFormat': 'h23', 'tradeDateBasis': 'close',
        'maxScreenshotsPerTrade': None})


def run():
    jwt = os.environ.get('TM_JWT_SECRET', '')
    if len(jwt) < 32 or jwt in ('dev-insecure-change-me', 'change-me'):
        raise RuntimeError('Set a strong TM_JWT_SECRET (Render generateValue)')
    port = int(os.environ.get('PORT', '10000'))
    if not 1024 <= port <= 65535 or port in (18080, 18081, 18964):
        raise RuntimeError('Invalid public PORT')
    children = []
    def shutdown(signum, frame):
        raise SystemExit(0)
    signal.signal(signal.SIGTERM, shutdown)
    signal.signal(signal.SIGINT, shutdown)
    # Never delete a caller-supplied path; every boot owns a fresh temporary DB.
    temporary = tempfile.TemporaryDirectory(prefix='tradelens-demo-')
    try:
        directory = Path(temporary.name)
        market = subprocess.Popen([sys.executable, str(SCRIPTS / 'showcase-market.py')])
        children.append(market)
        bootstrap = subprocess.Popen(['/server'], env=api_env(directory, secrets.token_hex(32), False))
        children.append(bootstrap)
        wait_api(bootstrap)
        # Wait for the fixture before the seed's news-validation calls.
        for attempt in range(50):
            try:
                get(MARKET_PROBE)
                break
            except OSError:
                if market.poll() is not None or attempt == 49:
                    raise RuntimeError('Fictional market provider failed')
                time.sleep(0.1)
        seed()
        stop(bootstrap)
        with sqlite3.connect(directory / 'demo.db') as conn:
            conn.execute('UPDATE users SET is_admin = 0')
            today = datetime.fromisoformat(os.environ['TRADELENS_SHOWCASE_TODAY']).replace(tzinfo=timezone.utc)
            # Include this week and adjacent weeks for calendar navigation; all events are fictional.
            for offset in range(-30, 15):
                day = today + timedelta(days=offset)
                for country, impact, hour, title in [('JPY', 'high', 1, 'Policy decision'),
                        ('USD', 'medium', 12, 'Employment survey'), ('EUR', 'low', 8, 'Price index')]:
                    stamp = day.replace(hour=hour).isoformat().replace('+00:00', 'Z')
                    conn.execute("INSERT INTO economic_events (provider,title,country,impact,event_ts,forecast,previous,actual,fetched_at) VALUES (?,?,?,?,?,?,?,?,?)",
                        ('FICTIONAL-demo', '[FICTIONAL demo] ' + title, country, impact, stamp,
                         '1.0%', '0.9%', '1.1%' if offset < 0 else '', today.isoformat()))
        api = subprocess.Popen(['/server'], env=api_env(directory, jwt, True))
        children.append(api)
        wait_api(api)

        class Ready(BaseHTTPRequestHandler):
            def do_GET(self):
                try:
                    if self.path != '/readyz' or any(p.poll() is not None for p in (market, api)):
                        raise RuntimeError('Not ready')
                    get('http://127.0.0.1:18080/healthz')
                    get(MARKET_PROBE)
                    with sqlite3.connect(directory / 'demo.db', timeout=1) as conn:
                        if conn.execute('SELECT count(*) FROM users').fetchone()[0] != 1:
                            raise RuntimeError('Invalid demo database')
                    self.send_response(200)
                except (OSError, ValueError, RuntimeError, sqlite3.Error):
                    self.send_response(503)
                self.end_headers()
            def log_message(self, *args):
                pass

        readiness = HTTPServer(('127.0.0.1', 18081), Ready)
        threading.Thread(target=readiness.serve_forever, daemon=True).start()
        entry = (ROOT / 'demo' / 'index.html').read_text().replace('__SHOWCASE_RANGE__', f'{SESSIONS[0]} – {SESSIONS[-1]}')
        entry_path = directory / 'entry.html'
        entry_path.write_text(entry)
        config = (ROOT / 'demo' / 'nginx.conf').read_text().replace('__PORT__', str(port)).replace('/app/demo/index.html', str(entry_path))
        config_path = directory / 'nginx.conf'
        config_path.write_text(config)
        proxy = subprocess.Popen(['nginx', '-c', str(config_path), '-g', 'daemon off;'])
        children.append(proxy)
        print('Fictional read-only demo initialized; starting public proxy', flush=True)
        while all(p.poll() is None for p in (market, api, proxy)):
            time.sleep(0.5)
        raise RuntimeError('Demo child process exited')
    finally:
        for child in reversed(children):
            stop(child)
        temporary.cleanup()


if __name__ == '__main__':
    if '--healthcheck' in sys.argv:
        with urllib.request.urlopen('http://127.0.0.1:' + os.environ.get('PORT', '10000') + '/readyz', timeout=4) as response:
            sys.exit(0 if response.status == 200 else 1)
    run()
