#!/usr/bin/env python3
"""Disposable Docker acceptance: python3 scripts/deployment-smoke.py [API image].
Uses synthetic fixtures only. Requires Docker and Python's standard library.
"""
import base64
from datetime import datetime, timedelta
import http.server
import json
import shutil
import subprocess
import sys
import tempfile
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path

IMAGE = sys.argv[1] if len(sys.argv) > 1 else "tradelens-api:local"
PREFIX = "tm-smoke-" + uuid.uuid4().hex[:8]
containers = []
calls = []


def docker(*args, data=None):
    return subprocess.check_output(["docker", *args], input=data)


def run(name, *args):
    containers.append(name)
    docker("run", "-d", "--name", name, *args)


class Bars(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        assert self.headers.get("Authorization") == "Bearer smoke-key"
        query = urllib.parse.parse_qs(urllib.parse.urlsplit(self.path).query)
        calls.append(query)
        stamp = datetime.fromisoformat(query["from"][0].replace("Z", "+00:00")) + timedelta(seconds=1)
        bar = dict(timestamp=stamp.isoformat(), open=1000, high=1000,
                   low=1000, close=1000, volume=0)
        bars = []
        until = datetime.fromisoformat(query["to"][0].replace("Z", "+00:00"))
        while stamp < until:
            bars.append(dict(bar, timestamp=stamp.isoformat()))
            stamp += timedelta(days=1)
        payload = dict(symbol=query["symbol"][0], interval=query["interval"][0],
                       source="synthetic-deployment-smoke", timezone="Asia/Tokyo",
                       adjustment_status="unadjusted", bars=bars)
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(payload).encode())

    def log_message(self, *_):
        pass


provider = http.server.ThreadingHTTPServer(("0.0.0.0", 0), Bars)
threading.Thread(target=provider.serve_forever, daemon=True).start()


def wait_ready(check):
    for _ in range(60):
        try:
            check()
            return
        except (OSError, subprocess.CalledProcessError):
            time.sleep(1)
    raise RuntimeError("service did not become ready")


def smoke(mode, root):
    data_dir = root / mode
    data_dir.mkdir()
    pg = PREFIX + "-pg"
    if mode == "postgres":
        run(pg, "-e", "POSTGRES_USER=tm", "-e", "POSTGRES_PASSWORD=smoke",
            "-e", "POSTGRES_DB=tm", "-p", "127.0.0.1::5432", "postgres:16")
        wait_ready(lambda: docker("exec", pg, "pg_isready", "-U", "tm"))
        pgport = docker("port", pg, "5432").decode().strip().split(":")[-1]
        db_url = f"postgres://tm:smoke@host.docker.internal:{pgport}/tm?sslmode=disable"
    else:
        db_url = "sqlite:///data/tm.db"
    name = PREFIX + "-" + mode
    run(name, "-p", "127.0.0.1::8080", "-v", f"{data_dir}:/data",
        "-e", "TM_JWT_SECRET=" + uuid.uuid4().hex + uuid.uuid4().hex,
        "-e", f"TM_DATABASE_URL={db_url}", "-e", "TM_ATTACH_DIR=/data/attachments",
        "-e", "TM_MARKET_DATA_ENABLED=true", "-e", "TM_MARKET_DATA_PROVIDER=http",
        "-e", f"TM_MARKET_DATA_HTTP_BASE_URL=http://host.docker.internal:{provider.server_port}",
        "-e", "TM_MARKET_DATA_HTTP_API_KEY=smoke-key", IMAGE)
    port = docker("port", name, "8080").decode().strip().split(":")[-1]
    base = f"http://127.0.0.1:{port}/api/v1"
    token = ""

    def request(method, path, body=None, status=200, content_type="application/json"):
        headers = {"Authorization": "Bearer " + token}
        if isinstance(body, dict):
            body = json.dumps(body).encode()
        if body is not None:
            headers["Content-Type"] = content_type
        req = urllib.request.Request(base + path, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=30) as response:
                raw = response.read()
                assert response.status == status, (path, response.status, raw)
                return json.loads(raw) if "json" in response.headers.get("Content-Type", "") else raw
        except urllib.error.HTTPError as err:
            raise AssertionError((method, path, err.code, err.read())) from err

    def upload(path, filename, raw, fields=None, mime="text/csv", status=200):
        boundary = uuid.uuid4().hex
        chunks = []
        for key, val in (fields or {}).items():
            chunks.append(f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{val}\r\n'.encode())
        chunks.extend([f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="{filename}"\r\nContent-Type: {mime}\r\n\r\n'.encode(), raw,
                       f'\r\n--{boundary}--\r\n'.encode()])
        return request("POST", path, b"".join(chunks), status, "multipart/form-data; boundary=" + boundary)

    wait_ready(lambda: urllib.request.urlopen(base + "/setup/status", timeout=2).close())
    credentials = dict(email="deployment@example.test", password="smoke-password-231")
    request("POST", "/setup", credentials, 201)
    token = request("POST", "/auth/login", credentials)["access_token"]
    account = request("POST", "/accounts", dict(name="SBI smoke", broker="SBI",
                      base_currency="JPY", starting_balance=1000000), 201)["id"]
    fill = dict(account_id=account, symbol="AAPL", instrument_type="stock",
                side="buy", quantity=10, price=200, executed_at="2026-09-15T09:59:00-04:00")
    created = request("POST", "/executions", fill, 201)
    fill["executed_at"] = "2026-09-15T13:59:00Z"
    assert request("POST", "/executions", fill)["execution_id"] == created["execution_id"]
    request("PATCH", "/executions/" + created["execution_id"],
            dict(side="buy", quantity=10, price=200, executed_at="2026-09-15T22:30:00+08:00"))
    detail = request("GET", "/trades/" + created["trade_id"])
    assert detail["opened_at"] == "2026-09-15T14:30:00Z", detail
    csv = "約定日,銘柄コード,取引,約定数量,約定単価,手数料/諸経費等\n2026/08/01,1515,信用新規買,100,1000,0\n2026/08/04,1515,現引,100,1090,500\n"
    imported = upload("/imports/commit", "sbi.csv", csv.encode(),
                      dict(account_id=account, column_mapping='{"symbol":"銘柄コード"}'))
    assert imported["inserted"] > 0, imported
    trades = request("GET", "/trades?account_id=" + account)
    converted = [tr for tr in trades if tr["symbol"] == "1515"]
    assert len(converted) == 2, trades
    assert next(tr for tr in converted if tr["status"] == "closed")["net_pnl"] == 0
    assert next(tr for tr in converted if tr["status"] == "open")["avg_entry_price"] == 1005
    png = base64.b64decode("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVQI12P4//8/AAX+Av7cxFnHAAAAAElFTkSuQmCC")
    attachment = upload("/trades/" + created["trade_id"] + "/attachments",
                        "smoke.png", png, mime="image/png", status=201)["id"]
    news = request("POST", "/news", dict(title="Synthetic deployment research",
                   source="manual", published_at="2026-09-20T17:30:00+08:00",
                   original_text="Synthetic fixture", notes="Persistence check",
                   assets=[dict(asset_type="stock", symbol="1515", market="JP")]), 201)
    request("POST", "/news/" + news["id"] + "/predictions",
            dict(news_asset_id=news["assets"][0]["id"], direction="bullish",
                 reasoning="Synthetic research", horizons=[1, 5]), 201)
    assert news["published_at"] == "2026-09-20T09:30:00Z", news
    paths = ["/news/" + news["id"], "/accounts", "/trades?account_id=" + account, "/news",
             "/trades/" + created["trade_id"], "/trades/" + created["trade_id"] + "/attachments",
             "/analytics/summary?account_id=" + account,
             "/analytics/account-value?account_id=" + account + "&from=2026-09-15&to=2026-09-15"]
    baseline = {path: request("GET", path) for path in paths}
    value = baseline[paths[-1]]
    assert value["points"] and value["points"][0]["estimated_account_value"] is not None, value
    assert calls, "HTTP provider must actually be called"
    market = request("GET", "/market/bars?symbol=AAPL&instrument_type=stock&interval=D&from=2026-09-18T00:00:00Z&to=2026-09-19T00:00:00Z")
    assert market["bars"], market

    def verify(stage):
        nonlocal token
        token = request("POST", "/auth/login", credentials)["access_token"]
        for path in paths:
            actual = request("GET", path)
            assert actual == baseline[path], (mode, stage, path, baseline[path], actual)
        assert request("GET", "/attachments/" + attachment + "/file") == png
        print(mode, stage, "PASS", flush=True)

    docker("restart", name)
    port = docker("port", name, "8080").decode().strip().split(":")[-1]
    base = f"http://127.0.0.1:{port}/api/v1"
    wait_ready(lambda: urllib.request.urlopen(base + "/setup/status", timeout=2).close())
    verify("restart+migration+persistence")
    docker("stop", name)
    backup = root / (mode + "-backup")
    shutil.copytree(data_dir, backup)
    if mode == "postgres":
        dump = docker("exec", pg, "pg_dump", "-U", "tm", "-d", "tm", "-Fc")
        docker("exec", pg, "createdb", "-U", "tm", "restored")
        docker("exec", "-i", pg, "pg_restore", "-U", "tm", "-d", "restored", "--exit-on-error", data=dump)
        # Replace the disposable original database with the restored one.
        docker("exec", pg, "dropdb", "-U", "tm", "tm")
        docker("exec", pg, "psql", "-U", "tm", "-d", "postgres", "-c", "ALTER DATABASE restored RENAME TO tm")
    else:
        (data_dir / "tm.db").unlink()
    # Restore database/attachment volume, including SQLite's checkpointed DB.
    shutil.copytree(backup, data_dir, dirs_exist_ok=True)
    docker("start", name)
    port = docker("port", name, "8080").decode().strip().split(":")[-1]
    base = f"http://127.0.0.1:{port}/api/v1"
    wait_ready(lambda: urllib.request.urlopen(base + "/setup/status", timeout=2).close())
    verify("backup+restore+migration")
    docker("stop", name)


try:
    with tempfile.TemporaryDirectory(prefix=PREFIX) as temp:
        for mode in ("sqlite", "postgres"):
            smoke(mode, Path(temp))
except Exception:
    for name in containers:
        subprocess.run(["docker", "logs", "--tail", "20", name], check=False)
    raise
finally:
    provider.shutdown()
    for name in reversed(containers):
        subprocess.run(["docker", "rm", "-fv", name], stdout=subprocess.DEVNULL, check=False)
