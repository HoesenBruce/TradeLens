#!/usr/bin/env python3
"""Container acceptance, including two clean boots. Build tradelens-demo:243 first."""
import json
import os
import secrets
import subprocess
import time
import urllib.error
import urllib.request

IMAGE = os.environ.get('DEMO_TEST_IMAGE', 'tradelens-demo:243')
BASE = 'http://127.0.0.1:19000'


def docker(*args):
    return subprocess.check_output(['docker', *args], text=True).strip()


def request(method, path, body=None, token='', expected=200):
    req = urllib.request.Request(BASE + path, method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={'Content-Type': 'application/json', 'Authorization': 'Bearer ' + token})
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            code, raw = response.status, response.read()
            if path in ('/', '/demo'):
                assert response.headers['Content-Type'].startswith('text/html')
    except urllib.error.HTTPError as error:
        code, raw = error.code, error.read()
    assert code == expected, (method, path, code, raw[:500])
    return json.loads(raw) if raw and raw[:1] in (b'{', b'[') else raw


def ready(container):
    start = time.monotonic()
    while time.monotonic() - start < 120:
        try:
            request('GET', '/readyz')
            print(f'ready in {time.monotonic() - start:.1f}s', flush=True)
            return
        except (OSError, AssertionError):
            if docker('inspect', '-f', '{{.State.Running}}', container) != 'true':
                raise AssertionError(docker('logs', container))
            time.sleep(0.5)
    raise AssertionError(docker('logs', container))


def check():
    request('GET', '/healthz')
    assert b'Try Demo' in request('GET', '/')
    assert b'Try Demo' in request('GET', '/demo')
    assert b'<html' in request('GET', '/home')
    status = request('GET', '/api/v1/setup/status')
    assert not status['needs_setup'] and not status['registration_open'] and status['user_count'] == 1
    request('GET', '/api/v1/accounts', expected=401)
    request('POST', '/api/v1/auth/login', {'email': 'demo@example.com', 'password': 'wrong'}, expected=401)
    tokens = request('POST', '/api/v1/auth/login', {'email': 'demo@example.com', 'password': 'fictional-demo-password'})
    refreshed = request('POST', '/api/v1/auth/refresh', {'refresh_token': tokens['refresh_token']})
    token = refreshed['access_token']
    me = request('GET', '/api/v1/me', token=token)
    assert not me['is_admin']
    prefs = request('GET', '/api/v1/me/preferences', token=token)['prefs']
    assert prefs['marketTimezone'] == 'Asia/Tokyo' and prefs['displayCurrency'] == 'JPY'
    accounts = request('GET', '/api/v1/accounts', token=token)
    assert len(accounts) == 1 and accounts[0]['base_currency'] == 'JPY'
    account = accounts[0]['id']
    trades = request('GET', '/api/v1/trades', token=token)
    assert len(trades) == 7 and all(t['qty_remaining'] == 0 for t in trades)
    assert sum(t['net_pnl'] for t in trades) == 25000
    values = request('GET', '/api/v1/analytics/account-value?account_id=' + account + '&from=2026-09-01&to=2026-09-08', token=token)
    assert len(values['points']) == 6 and all(p['status'] == 'complete' for p in values['points']), values
    assert values['points'][-1]['estimated_account_value'] == 975000, values
    news = request('GET', '/api/v1/news', token=token)
    assert len(news) == 3
    counts = request('GET', '/api/v1/news/performance', token=token)['counts']
    assert counts['total'] == 6 and counts['unavailable'] == 2 and counts['pending'] == 4, counts
    prediction_count = 0
    for item in news:
        detail = request('GET', '/api/v1/news/' + item['id'], token=token)
        assert detail['title'].startswith('[FICTIONAL')
        prediction_count += len(detail['predictions'])
        for prediction in detail['predictions']:
            request('GET', '/api/v1/news/' + item['id'] + '/predictions/' + prediction['id'] + '/validations', token=token)
    assert prediction_count == 3
    for method, path in [
        ('POST', '/setup'), ('POST', '/auth/register'), ('POST', '/accounts'),
        ('DELETE', '/accounts/' + account), ('PATCH', '/trades/' + trades[0]['id']),
        ('PUT', '/me/password'), ('POST', '/me/totp/start'),
        ('GET', '/access-tokens'), ('POST', '/access-tokens'), ('DELETE', '/access-tokens/x'),
        ('POST', '/imports'), ('POST', '/imports/commit'), ('POST', '/media'),
        ('POST', '/trades/' + trades[0]['id'] + '/attachments'),
        ('POST', '/ocr'), ('GET', '/admin/users'), ('POST', '/admin/users'),
        ('PUT', '/settings/coach'), ('POST', '/settings/ocr/test'),
        ('POST', '/news/' + news[0]['id'] + '/analyze'), ('POST', '/share-links'),
        ('GET', '/flex-sync'), ('GET', '/attachments/x/file')]:
        request(method, '/api/v1' + path, {}, token, expected=403)
    features = request('GET', '/api/v1/system/info', token=token)['features']
    assert features['market_data'] and not any(features[k] for k in ('ocr', 'coach', 'share_links', 'econ_calendar', 'background_jobs'))
    assert len(request('GET', '/api/v1/trades', token=token)) == 7
    return me['id'], token


invalid = docker('run', '-d', IMAGE)
try:
    result = subprocess.run(['docker', 'wait', invalid], capture_output=True, text=True, timeout=20, check=True)
    assert result.stdout.strip() != '0', 'missing JWT secret must fail startup'
finally:
    docker('rm', '-f', invalid)

container = docker('run', '-d', '--memory=512m', '--cpus=0.5', '-p', '127.0.0.1:19000:10000',
                   '-e', 'TM_JWT_SECRET=' + secrets.token_hex(32),
                   '-e', 'TM_DATABASE_URL=postgres://must-never-be-used/production',
                   '-e', 'TM_ALLOW_REGISTRATION=true', '-e', 'TM_COACH_ENABLED=true', IMAGE)
try:
    ready(container)
    first, token = check()
    # The backend itself rejects writes; the reverse proxy is not the safety boundary.
    raw = docker('exec', container, 'python3', '-c',
        "import urllib.request, urllib.error\nr=urllib.request.Request('http://127.0.0.1:18080/api/v1/me/password', data=b'{}', method='PUT', headers={'Authorization':'Bearer " + token + "'})\ntry: urllib.request.urlopen(r); raise AssertionError('write allowed')\nexcept urllib.error.HTTPError as e: assert e.code == 403")
    docker('restart', container)
    ready(container)
    second, _ = check()
    assert first != second, 'restart must recreate the fictional database'
    request('GET', '/api/v1/me', token=token, expected=401)
    print(docker('stats', '--no-stream', '--format', '{{.MemUsage}}', container))
    docker('exec', container, 'python3', '-c',
        "from pathlib import Path\nimport os, signal\nfor p in Path('/proc').glob('[0-9]*/cmdline'):\n try:\n  if p.read_bytes().endswith(b'/app/scripts/showcase-market.py\\x00'): os.kill(int(p.parent.name), signal.SIGTERM)\n except (FileNotFoundError, ProcessLookupError): pass")
    result = subprocess.run(['docker', 'wait', container], capture_output=True, text=True, timeout=20, check=True)
    assert result.stdout.strip() != '0', 'provider death must fail the whole service'
    print('demo initialization, reset, auth, permissions, SBI and news checks passed')
finally:
    docker('rm', '-f', container)
