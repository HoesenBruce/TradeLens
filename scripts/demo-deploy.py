#!/usr/bin/env python3
"""Release-gated Render deployment. Standard library only; never log API bodies."""
import base64
import datetime
import io
import json
import os
from pathlib import Path
import re
import subprocess
import time
import urllib.error
import urllib.request
import zipfile

REPO = 'HoesenBruce/TradeLens'
ORIGIN = 'https://tradelens-demo.onrender.com'


class DeploymentError(ValueError):
    pass


PENDING = {'created', 'queued', 'build_in_progress', 'pre_deploy_in_progress', 'update_in_progress'}


def require(condition, message):
    if not condition:
        raise DeploymentError(message)


def gh(path, binary=False):
    raw = subprocess.check_output(['gh', 'api', f'repos/{REPO}/' + path], stderr=subprocess.DEVNULL, timeout=60)
    return raw if binary else json.loads(raw)


def git(*args):
    return subprocess.check_output(['git', *args], text=True, stderr=subprocess.DEVNULL, timeout=30).strip()


def ancestor(older, newer):
    return gh(f'compare/{older}...{newer}')['status'] in ('ahead', 'identical')


def release_source(version, expected=''):
    require(bool(re.fullmatch(r'(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)', version)), 'Only stable semver versions are accepted')
    release = gh(f'releases/tags/v{version}')
    require(not release['draft'] and not release['prerelease'] and release['tag_name'] == f'v{version}', 'Release must be published and stable')
    require(gh('releases/latest')['id'] == release['id'], 'Superseded release: only latest stable may deploy')
    obj = gh(f'git/ref/tags/v{version}')['object']
    # Support annotated tags without trusting the mutable target_commitish field.
    for _ in range(5):
        if obj['type'] == 'commit':
            break
        require(obj['type'] == 'tag', 'Invalid tag object')
        obj = gh('git/tags/' + obj['sha'])['object']
    sha = obj['sha']
    require(obj['type'] == 'commit' and bool(re.fullmatch('[0-9a-f]{40}', sha)), 'Tag must resolve to a full commit SHA')
    require(not expected or expected == sha, 'Release tag moved or source SHA mismatched')
    require(ancestor(sha, 'main'), 'Release commit must be on main')
    require(git('show', f'{sha}:VERSION') == version, 'Release VERSION mismatch')
    require(json.loads(git('show', f'{sha}:.release-please-manifest.json')).get('.') == version, 'Release manifest mismatch')
    contract = json.loads(git('show', f'{sha}:deploy/demo/smoke.json'))
    require(contract['schema'] == 1, 'Unsupported release smoke contract')
    git('cat-file', '-e', f'{sha}:deploy/demo/Dockerfile')
    return sha, contract


def http(url, method='GET', body=None, headers=None):
    request = urllib.request.Request(url, method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={'Content-Type': 'application/json', **(headers or {})})
    # Refuse redirects: credentials must never follow a server-supplied location.
    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, *args, **kwargs):
            return None
    try:
        with urllib.request.build_opener(NoRedirect).open(request, timeout=30) as response:
            return response.status, response.read(), response.headers
    except urllib.error.HTTPError as error:
        return error.code, error.read(), error.headers


def publication(run_id, version, sha, automatic=False):
    require(bool(re.fullmatch('[0-9]+', run_id)), 'Invalid publication run ID')
    run = gh('actions/runs/' + run_id)
    require(run['path'] in ('.github/workflows/release-please.yml', '.github/workflows/docker-publish.yml'), 'Untrusted publication workflow')
    require(run['event'] in ('push', 'workflow_dispatch'), 'Untrusted publication event')
    require(run['status'] == 'completed' or
            (automatic and run_id == os.environ['GITHUB_RUN_ID'] and run['status'] == 'in_progress'), 'Publication run must be complete or the current automatic parent')
    require(run['head_sha'] == sha and run['head_branch'] == ('main' if run['event'] == 'push' else 'v' + version), 'Publication run must target this release commit on main or its exact tag')
    jobs = gh(f'actions/runs/{run_id}/jobs?filter=all&per_page=100')['jobs']
    for image in ('api', 'web'):
        candidates = [j for j in jobs if j['name'] == f'Push {image}' or j['name'].endswith(f' / Push {image}')]
        require(bool(candidates), f'Missing {image} publishing job')
        latest_job = max(candidates, key=lambda j: j['id'])
        require(latest_job['status'] == 'completed' and latest_job['conclusion'] == 'success', f'{image} publication must have succeeded')
    artifacts = gh(f'actions/runs/{run_id}/artifacts?per_page=100')['artifacts']
    for image in ('api', 'web'):
        matches = [a for a in artifacts if a['name'] == f'ghcr-digest-{image}-{sha}' and not a['expired']]
        require(len(matches) == 1, f'Missing unambiguous {image} publication evidence')
        with zipfile.ZipFile(io.BytesIO(gh(f'actions/artifacts/{matches[0]["id"]}/zip', binary=True))) as archive:
            record = archive.read('image-digest.txt').decode().strip()
        match = re.fullmatch(rf'ghcr.io/hoesenbruce/tradelens-{image}@(sha256:[0-9a-f]{{64}}) source={sha} version={re.escape(version)}', record)
        require(match is not None, f'{image} publication source/version mismatch')
        # Both immutable registry tags must still point at the publisher's digest.
        status, raw, _ = http(f'https://ghcr.io/token?service=ghcr.io&scope=repository:hoesenbruce/tradelens-{image}:pull')
        require(status == 200, 'GHCR read token unavailable')
        token = json.loads(raw)['token']
        for tag in (version, 'sha-' + sha):
            status, _, headers = http(f'https://ghcr.io/v2/hoesenbruce/tradelens-{image}/manifests/{tag}', headers={
                'Authorization': 'Bearer ' + token,
                'Accept': 'application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json'})
            require(status == 200 and headers.get('Docker-Content-Digest') == match[1], f'{image} registry digest mismatch')


class Render:
    def __init__(self, service, key):
        require(bool(re.fullmatch('srv-[a-z0-9]{20}', service)), 'Configure valid RENDER_SERVICE_ID')
        require(bool(key), 'Configure RENDER_API_KEY in render-demo environment')
        self.base = 'https://api.render.com/v1/services/' + service
        self.key = key

    def api(self, path='', method='GET', body=None):
        status, raw, _ = http(self.base + path, method, body, {'Authorization': 'Bearer ' + self.key})
        require(status in (200, 201, 202), f'Render API {method} failed (HTTP {status}); inspect Dashboard, no blind retry')
        return json.loads(raw)

    def service_check(self):
        service = self.api()
        details = service['serviceDetails']
        docker = details['envSpecificDetails']
        require(service['name'] == 'tradelens-demo' and service['type'] == 'web_service' and
                service['repo'].removesuffix('.git') == f'https://github.com/{REPO}' and service['branch'] == 'main' and
                service['rootDir'] in ('', '.') and service['suspended'] == 'not_suspended', 'Unexpected Render service/source')
        require(service.get('autoDeployTrigger') == 'off' and service['autoDeploy'] == 'no', 'Render Auto-Deploy must stay Off')
        require(details['runtime'] == 'docker' and details['plan'] == 'free' and details['numInstances'] == 1 and
                not details.get('disk') and not details.get('autoscaling') and details['url'] == ORIGIN and
                details['healthCheckPath'] == '/readyz', 'Require existing single Free service without disk/autoscaling')
        require(docker['dockerfilePath'].removeprefix('./') == 'deploy/demo/Dockerfile' and
                docker['dockerContext'] in ('', '.') and not docker['dockerCommand'] and not docker.get('preDeployCommand'), 'Unexpected demo Docker configuration')

    def live(self):
        rows = self.api('/deploys?status=live&limit=2')
        require(len(rows) == 1, 'Expected exactly one Live deploy')
        deploy = rows[0]['deploy']
        require(deploy['status'] == 'live' and bool(re.fullmatch('[0-9a-f]{40}', deploy['commit']['id'])), 'Cannot establish Live source SHA')
        return deploy

    def idle(self):
        # A timed-out workflow must not queue another deployment over an ongoing build.
        for state in sorted(PENDING):
            require(not self.api(f'/deploys?status={state}&limit=1'), 'Existing Render deployment in progress; wait and inspect before retry')


def smoke(contract):
    def request(method, path, body=None, token='', expected=200):
        status, raw, _ = http(ORIGIN + path, method, body, {'Authorization': 'Bearer ' + token} if token else {})
        require(status == expected, f'Smoke {method} {path.split("?")[0]} failed (HTTP {status})')
        return json.loads(raw) if raw[:1] in (b'[', b'{') else raw
    deadline = time.monotonic() + 300
    while True:
        try:
            request('GET', '/readyz')
            break
        except (OSError, ValueError):
            require(time.monotonic() < deadline, 'Public readiness timed out')
            time.sleep(10)
    require(b'Try Demo' in request('GET', '/demo'), 'Demo entry page missing')
    request('GET', '/api/v1/accounts', expected=401)
    tokens = request('POST', '/api/v1/auth/login', {'email': 'demo@example.com', 'password': 'fictional-demo-password'})
    token = tokens['access_token']
    require(not request('GET', '/api/v1/me', token=token)['is_admin'], 'Demo user must not be admin')
    accounts = request('GET', '/api/v1/accounts', token=token)
    require(len(accounts) == 1 and accounts[0]['base_currency'] == 'JPY', 'Fictional JPY account missing')
    account = accounts[0]['id']
    trades = request('GET', '/api/v1/trades', token=token)
    require(len(trades) == contract['trades'] and all(t['qty_remaining'] == 0 for t in trades) and
            sum(t['net_pnl'] for t in trades) == contract['net_pnl'], 'Fictional trade/P&L mismatch')
    require(len(request('GET', '/api/v1/executions?account_id=' + account, token=token)) == contract['executions'], 'Execution count mismatch')
    # End at the actual last fixture close, independent of deploy/startup midnight.
    end = max(t['closed_at'][:10] for t in trades)
    points = request('GET', '/api/v1/analytics/account-value?account_id=' + account + '&to=' + end, token=token)['points']
    require(bool(points) and points[-1]['status'] == 'complete' and points[-1]['estimated_account_value'] == contract['account_value'] and
            points[-1]['contributed_capital'] == contract['contributed_capital'], 'Fictional account value/capital mismatch')
    news = request('GET', '/api/v1/news', token=token)
    require(len(news) == contract['news'] and all(n['title'].startswith('[FICTIONAL') for n in news), 'Fictional news missing')
    following = (datetime.date.fromisoformat(end) + datetime.timedelta(days=1)).isoformat()
    calendar = request('GET', f'/api/v1/economic-events?from={end}&to={following}', token=token)
    require(bool(calendar) and all(e['provider'] == 'FICTIONAL-demo' for e in calendar), 'Fictional calendar missing')
    for method, path in [('POST', '/accounts'), ('DELETE', '/accounts/' + account), ('POST', '/imports'),
                         ('PUT', '/me/password'), ('GET', '/access-tokens'), ('GET', '/admin/users'),
                         ('POST', '/share-links'), ('POST', '/news/' + news[0]['id'] + '/analyze')]:
        status, raw, _ = http(ORIGIN + '/api/v1' + path, method, {}, {'Authorization': 'Bearer ' + token})
        require(status == 403 and json.loads(raw).get('error', {}).get('code') == 'demo_read_only', 'Backend read-only guard failed')
    require(request('GET', '/api/v1/trades', token=token) == trades, 'Smoke changed fictional trades')


def deploy(render, version, sha, contract, revalidate, record, bootstrap=''):
    render.service_check()
    render.idle()
    live = render.live()
    record['last_verified_live_sha'] = live['commit']['id']
    if not ancestor(live['commit']['id'], sha):
        require(bootstrap and bootstrap == live['commit']['id'], 'Live source is newer/divergent; manual first-deploy approval required')
        live_version = base64.b64decode(gh(f'contents/VERSION?ref={bootstrap}')['content']).decode().strip()
        require(bool(re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+', live_version)) and
                tuple(map(int, live_version.split('.'))) < tuple(map(int, version.split('.'))), 'Bootstrap may only replace an older-version demo')
    # Recheck after approval and all publication/service probes, immediately before POST.
    revalidate()
    current = render.live()
    require(current['id'] == live['id'], 'Live deployment changed during validation')
    render.idle()
    result = render.api('/deploys', 'POST', {'commitId': sha, 'clearCache': 'do_not_clear'})
    deploy_id = result['id']
    require(bool(re.fullmatch('dep-[a-z0-9]{20}', deploy_id)), 'Invalid Render deploy ID')
    record['deploy_id'] = deploy_id
    deadline = time.monotonic() + 1800
    previous = None
    while True:
        result = render.api('/deploys/' + deploy_id)
        state = result['status']
        require(state in PENDING | {'live', 'deactivated', 'build_failed', 'update_failed', 'canceled', 'pre_deploy_failed'}, 'Unknown Render deployment state')
        record['render_state'] = state
        if state != previous:
            print('Render state: ' + state, flush=True)
            previous = state
        if state == 'live':
            require(result['commit']['id'] == sha, 'Render deploy source mismatch')
            break
        require(state in PENDING, 'Render deployment failed: ' + state)
        require(time.monotonic() < deadline, 'Render deployment timed out; it may still be running')
        time.sleep(15)
    require(render.live()['id'] == deploy_id and render.live()['commit']['id'] == sha, 'Actual Live deployment mismatch')
    record['last_verified_live_sha'] = sha
    smoke(contract)
    render.service_check()
    require(render.live()['id'] == deploy_id, 'Live deployment changed during smoke')
    record.update(status='success', smoke='passed')


def main():
    record = {'status': 'failed', 'started_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
              'last_verified_live_sha': 'unknown', 'smoke': 'not passed'}
    render = None
    try:
        event = os.environ['GITHUB_EVENT_NAME']
        require(os.environ['GITHUB_REPOSITORY'] == REPO and os.environ['GITHUB_REF'] == 'refs/heads/main' and
                event in ('push', 'workflow_dispatch'), 'Only trusted main release chain or manual dispatch accepted')
        version = os.environ['INPUT_VERSION']
        sha, contract = release_source(version, os.environ.get('EXPECTED_SOURCE_SHA', ''))
        run_id = os.environ['PUBLICATION_RUN_ID']
        record.update(version=version, source_sha=sha, publication_run_id=run_id)
        publication(run_id, version, sha, automatic=event == 'push')
        render = Render(os.environ.get('RENDER_SERVICE_ID', ''), os.environ.get('RENDER_API_KEY', ''))
        def revalidate():
            release_source(version, sha)
            publication(run_id, version, sha, automatic=event == 'push')
        deploy(render, version, sha, contract, revalidate, record,
               os.environ.get('BOOTSTRAP_LIVE_SHA', '') if event == 'workflow_dispatch' else '')
    except Exception as error:
        # Never include upstream response bodies/exception strings, URLs or tokens.
        record['error'] = str(error) if isinstance(error, DeploymentError) else 'Unexpected validation/API/smoke error; inspect provider run and Dashboard without printing credentials'
        print('Demo deployment FAILED: ' + record['error'], flush=True)
        if render:
            try:
                record['last_verified_live_sha'] = render.live()['commit']['id']
            except Exception:
                pass
        raise
    finally:
        record['finished_at'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        Path('demo-deploy-record.json').write_text(json.dumps(record, indent=2) + '\n')
        with open(os.environ['GITHUB_STEP_SUMMARY'], 'a') as summary:
            summary.write('```json\n' + json.dumps(record, indent=2) + '\n```\n')


if __name__ == '__main__':
    try:
        main()
    except Exception:
        print('Demo deployment FAILED. See sanitized record, publication run and Render Dashboard; no automatic rollback.')
        raise SystemExit(1)
