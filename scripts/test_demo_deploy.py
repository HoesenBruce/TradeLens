import copy
import importlib.util
import io
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile

spec = importlib.util.spec_from_file_location('demo_deploy', Path(__file__).with_name('demo-deploy.py'))
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)
SHA = 'a' * 40
OLD = 'b' * 40
DEP = 'dep-' + 'a' * 20
CONTRACT = json.loads(Path('deploy/demo/smoke.json').read_text())
SERVICE = {
    'name': 'tradelens-demo', 'type': 'web_service', 'repo': 'https://github.com/HoesenBruce/TradeLens',
    'branch': 'main', 'rootDir': '', 'suspended': 'not_suspended', 'autoDeployTrigger': 'off', 'autoDeploy': 'no',
    'serviceDetails': {'runtime': 'docker', 'plan': 'free', 'numInstances': 1, 'url': m.ORIGIN,
        'healthCheckPath': '/readyz', 'envSpecificDetails': {'dockerfilePath': './deploy/demo/Dockerfile',
            'dockerContext': '.', 'dockerCommand': ''}}}


class Deployment(unittest.TestCase):
    def render(self, states=('build_in_progress', 'live'), live_sha=OLD):
        render = m.Render('srv-' + 'a' * 20, 'mock-key')
        queue = iter(states)
        render.created = False
        render.posts = []
        def api(path='', method='GET', body=None):
            if not path:
                return copy.deepcopy(SERVICE)
            if method == 'POST':
                render.created = True
                render.posts.append(body)
                return {'id': DEP}
            if path.startswith('/deploys?status=live'):
                return [{'deploy': {'id': DEP if render.created else 'dep-' + 'b' * 20,
                    'status': 'live', 'commit': {'id': SHA if render.created else live_sha}}}]
            if path.startswith('/deploys?status='):
                return []
            return {'id': DEP, 'status': next(queue), 'commit': {'id': SHA}}
        render.api = api
        return render

    def test_exact_sha_and_completion(self):
        render = self.render()
        record = {}
        calls = []
        with patch.object(m, 'ancestor', return_value=True), patch.object(m, 'smoke') as smoke, patch.object(m.time, 'sleep'):
            m.deploy(render, '1.2.3', SHA, CONTRACT, lambda: calls.append('validate'), record)
        self.assertEqual(render.posts, [{'commitId': SHA, 'clearCache': 'do_not_clear'}])
        self.assertEqual(record['status'], 'success')
        self.assertEqual(record['last_verified_live_sha'], SHA)
        self.assertEqual(calls, ['validate'])
        smoke.assert_called_once_with(CONTRACT)

    def test_failures_never_succeed(self):
        for states in [('build_failed',), ('update_failed',), ('canceled',), ('deactivated',), ('unknown',)]:
            with self.subTest(states=states), patch.object(m, 'ancestor', return_value=True), patch.object(m, 'smoke') as smoke:
                record = {}
                with self.assertRaises(ValueError):
                    m.deploy(self.render(states), '1.2.3', SHA, CONTRACT, lambda: None, record)
                self.assertNotIn('status', record)
                smoke.assert_not_called()
        with patch.object(m, 'ancestor', return_value=True), patch.object(m, 'smoke', side_effect=ValueError('bad smoke')):
            record = {}
            with self.assertRaises(ValueError):
                m.deploy(self.render(('live',)), '1.2.3', SHA, CONTRACT, lambda: None, record)
            self.assertEqual(record['last_verified_live_sha'], SHA)
            self.assertNotIn('status', record)

    def test_timeout_mismatch_and_active_build(self):
        with patch.object(m, 'ancestor', return_value=True), patch.object(m.time, 'monotonic', side_effect=[0, 1801]):
            with self.assertRaisesRegex(ValueError, 'timed out'):
                m.deploy(self.render(('queued',)), '1.2.3', SHA, CONTRACT, lambda: None, {})
        render = self.render(('live',))
        original = render.api
        def mismatch(path='', method='GET', body=None):
            result = original(path, method, body)
            if path == '/deploys/' + DEP:
                result['commit']['id'] = OLD
            return result
        render.api = mismatch
        with patch.object(m, 'ancestor', return_value=True), self.assertRaisesRegex(ValueError, 'source mismatch'):
            m.deploy(render, '1.2.3', SHA, CONTRACT, lambda: None, {})
        render = self.render()
        original = render.api
        render.api = lambda path='', method='GET', body=None: [{'deploy': {}}] if 'status=queued' in path else original(path, method, body)
        with self.assertRaisesRegex(ValueError, 'in progress'):
            m.deploy(render, '1.2.3', SHA, CONTRACT, lambda: None, {})
        self.assertEqual(render.posts, [])

    def test_newer_live_and_late_release_change(self):
        for bootstrap in ('', SHA):
            render = self.render()
            with patch.object(m, 'ancestor', return_value=False), self.assertRaisesRegex(ValueError, 'newer/divergent'):
                m.deploy(render, '1.2.3', SHA, CONTRACT, lambda: None, {}, bootstrap)
            self.assertEqual(render.posts, [])
        with patch.object(m, 'ancestor', return_value=False), patch.object(m, 'smoke'), patch.object(m, 'gh', return_value={'content': 'MC4yLjE='}):
            m.deploy(self.render(('live',)), '1.2.3', SHA, CONTRACT, lambda: None, {}, OLD)
        with patch.object(m, 'ancestor', return_value=False), patch.object(m, 'gh', return_value={'content': 'Mi4wLjA='}), self.assertRaisesRegex(ValueError, 'older-version'):
            m.deploy(self.render(('live',)), '1.2.3', SHA, CONTRACT, lambda: None, {}, OLD)
        render = self.render()
        def changed():
            raise ValueError('release changed after approval')
        with patch.object(m, 'ancestor', return_value=True), self.assertRaises(ValueError):
            m.deploy(render, '1.2.3', SHA, CONTRACT, changed, {})
        self.assertEqual(render.posts, [])

    def test_service_guard_and_http_sanitization(self):
        mutations = [({'branch': 'feature'}, None), ({'autoDeployTrigger': 'commit'}, None),
                     ({}, {'plan': 'starter'}), ({}, {'disk': {'id': 'disk'}}),
                     ({}, {'numInstances': 2}), ({}, {'runtime': 'image'}),
                     ({}, {'url': 'https://other.example'})]
        for top, detail in mutations:
            service = copy.deepcopy(SERVICE)
            service.update(top)
            service['serviceDetails'].update(detail or {})
            with patch.object(m.Render, 'api', return_value=service), self.assertRaises(ValueError):
                m.Render('srv-' + 'a' * 20, 'mock').service_check()
        with patch.object(m, 'http', return_value=(401, b'secret response', {})):
            with self.assertRaisesRegex(ValueError, 'HTTP 401') as ctx:
                m.Render('srv-' + 'a' * 20, 'mock').api()
            self.assertNotIn('secret', str(ctx.exception))

    def test_release_validation(self):
        release = {'id': 1, 'tag_name': 'v1.2.3', 'draft': False, 'prerelease': False}
        def api(path):
            if path.startswith('git/ref'):
                return {'object': {'type': 'commit', 'sha': SHA}}
            if path.startswith('compare/'):
                return {'status': 'ahead'}
            return release
        def git(*args):
            value = args[-1]
            if value.endswith(':VERSION'):
                return '1.2.3'
            if value.endswith(':.release-please-manifest.json'):
                return '{".":"1.2.3"}'
            if value.endswith(':deploy/demo/smoke.json'):
                return json.dumps(CONTRACT)
            return ''
        with patch.object(m, 'gh', side_effect=api), patch.object(m, 'git', side_effect=git):
            self.assertEqual(m.release_source('1.2.3', SHA), (SHA, CONTRACT))
            for version in ['1.2.3-rc.1', '01.2.3', 'latest', '1.2.3\n', '../main']:
                with self.assertRaises(ValueError):
                    m.release_source(version)
            with self.assertRaisesRegex(ValueError, 'moved'):
                m.release_source('1.2.3', OLD)
            for key in ('draft', 'prerelease'):
                release[key] = True
                with self.assertRaises(ValueError):
                    m.release_source('1.2.3')
                release[key] = False
        with patch.object(m, 'gh', side_effect=lambda p: {'id': 2} if p == 'releases/latest' else api(p)), self.assertRaisesRegex(ValueError, 'Superseded'):
            m.release_source('1.2.3')
        with patch.object(m, 'gh', side_effect=api), patch.object(m, 'ancestor', return_value=False), self.assertRaisesRegex(ValueError, 'on main'):
            m.release_source('1.2.3')

    def test_publication_both_images_and_registry(self):
        digest = 'sha256:' + 'c' * 64
        run = {'path': '.github/workflows/docker-publish.yml', 'event': 'workflow_dispatch', 'status': 'completed', 'conclusion': 'success', 'head_sha': SHA, 'head_branch': 'v1.2.3'}
        jobs = [{'id': i, 'name': 'Push ' + image, 'status': 'completed', 'conclusion': 'success'} for i, image in enumerate(('api', 'web'))]
        artifacts = [{'id': i, 'name': f'ghcr-digest-{image}-{SHA}', 'expired': False} for i, image in enumerate(('api', 'web'))]
        def api(path, binary=False):
            if '/jobs?' in path:
                return {'jobs': jobs}
            if path.endswith('/artifacts?per_page=100'):
                return {'artifacts': artifacts}
            if path.endswith('/zip'):
                image = 'api' if '/0/' in path else 'web'
                data = io.BytesIO()
                with zipfile.ZipFile(data, 'w') as archive:
                    archive.writestr('image-digest.txt', f'ghcr.io/hoesenbruce/tradelens-{image}@{digest} source={SHA} version=1.2.3\n')
                return data.getvalue()
            return run
        def http(url, **kwargs):
            return (200, b'{"token":"mock"}', {}) if '/token?' in url else (200, b'{}', {'Docker-Content-Digest': digest})
        with patch.object(m, 'gh', side_effect=api), patch.object(m, 'http', side_effect=http):
            m.publication('123', '1.2.3', SHA)
            for conclusion in ('failure', 'cancelled', None):
                jobs[1]['conclusion'] = conclusion
                with self.assertRaises(ValueError):
                    m.publication('123', '1.2.3', SHA)
            jobs[1]['conclusion'] = 'success'
            # Demo-only failure does not erase successful publication evidence.
            run['conclusion'] = 'failure'
            m.publication('123', '1.2.3', SHA)
            run['conclusion'] = 'success'
            run['head_branch'] = 'feature'
            with self.assertRaisesRegex(ValueError, 'exact tag'):
                m.publication('123', '1.2.3', SHA)
            run['head_branch'] = 'v1.2.3'
            run['head_sha'] = OLD
            with self.assertRaisesRegex(ValueError, 'release commit'):
                m.publication('123', '1.2.3', SHA)
            run['head_sha'] = SHA
            run.update(path='.github/workflows/release-please.yml', event='push', head_branch='main', status='in_progress', conclusion=None)
            with patch.dict(os.environ, {'GITHUB_RUN_ID': '123'}):
                m.publication('123', '1.2.3', SHA, automatic=True)
                with self.assertRaises(ValueError):
                    m.publication('123', '1.2.3', SHA)
                with self.assertRaises(ValueError):
                    m.publication('456', '1.2.3', SHA, automatic=True)
            run.update(status='completed', conclusion='success')
            with patch.object(m, 'http', return_value=(200, b'{"token":"mock"}', {'Docker-Content-Digest': 'wrong'})), self.assertRaisesRegex(ValueError, 'digest mismatch'):
                m.publication('123', '1.2.3', SHA)
            artifacts.pop()
            with self.assertRaisesRegex(ValueError, 'web'):
                m.publication('123', '1.2.3', SHA)
        with patch.object(m, 'gh', side_effect=api), patch.object(m, 'http', return_value=(404, b'', {})), self.assertRaises(ValueError):
            m.publication('123', '1.2.3', SHA)

    def test_failure_record_redacts_credentials(self):
        env = {'GITHUB_EVENT_NAME': 'push', 'GITHUB_REPOSITORY': m.REPO, 'GITHUB_REF': 'refs/heads/main',
               'INPUT_VERSION': '1.2.3', 'PUBLICATION_RUN_ID': '123', 'RENDER_SERVICE_ID': 'srv-' + 'a' * 20,
               'RENDER_API_KEY': 'private-token'}
        original = Path.cwd()
        with tempfile.TemporaryDirectory() as temp, patch.dict(os.environ, env):
            try:
                os.chdir(temp)
                os.environ['GITHUB_STEP_SUMMARY'] = str(Path(temp) / 'summary')
                with patch.object(m, 'release_source', return_value=(SHA, CONTRACT)), patch.object(m, 'publication'), patch.object(m, 'deploy', side_effect=ValueError('private-token')), patch.object(m.Render, 'live', return_value={'commit': {'id': OLD}}):
                    with self.assertRaises(ValueError):
                        m.main()
                record = Path('demo-deploy-record.json').read_text()
                self.assertNotIn('private-token', record)
                self.assertEqual(json.loads(record)['last_verified_live_sha'], OLD)
                self.assertEqual(json.loads(record)['status'], 'failed')
            finally:
                os.chdir(original)

    def test_full_mock_render_api_and_public_smoke(self):
        created = False
        seen = []
        trades = [{'id': 't', 'qty_remaining': 0, 'net_pnl': 25000, 'closed_at': '2026-10-08T06:00:00Z'}] * 107
        trades[0] = dict(trades[0], net_pnl=25000 - 106 * 25000)
        def http(url, method='GET', body=None, headers=None):
            nonlocal created
            seen.append((url, method, body, headers))
            if url.startswith('https://api.render.com'):
                self.assertEqual(headers['Authorization'], 'Bearer mock-key')
                path = url.split('srv-' + 'a' * 20)[1]
                if not path:
                    result = SERVICE
                elif method == 'POST':
                    self.assertEqual(body, {'commitId': SHA, 'clearCache': 'do_not_clear'})
                    created = True
                    result = {'id': DEP}
                elif 'status=live' in path:
                    result = [{'deploy': {'id': DEP if created else 'old', 'status': 'live',
                        'commit': {'id': SHA if created else OLD}}}]
                elif 'status=' in path:
                    result = []
                else:
                    result = {'id': DEP, 'status': 'live', 'commit': {'id': SHA}}
                return 200, json.dumps(result).encode(), {}
            path = url.removeprefix(m.ORIGIN)
            if path == '/readyz':
                result = {'ready': True}
            elif path == '/demo':
                return 200, b'<html>Try Demo</html>', {}
            elif path == '/api/v1/auth/login':
                result = {'access_token': 'mock-session'}
            elif not headers:
                return 401, b'', {}
            elif method != 'GET' or path in ('/api/v1/access-tokens', '/api/v1/admin/users'):
                return 403, b'{"error":{"code":"demo_read_only"}}', {}
            elif path == '/api/v1/me':
                result = {'is_admin': False}
            elif path == '/api/v1/accounts':
                result = [{'id': 'account', 'base_currency': 'JPY'}]
            elif path == '/api/v1/trades':
                result = trades
            elif path.startswith('/api/v1/executions'):
                result = [{}] * 214
            elif path.startswith('/api/v1/analytics/account-value'):
                result = {'points': [{'status': 'complete', 'estimated_account_value': 975000, 'contributed_capital': 950000}]}
            elif path == '/api/v1/news':
                result = [{'id': 'news', 'title': '[FICTIONAL] example'}] * 3
            else:
                result = [{'provider': 'FICTIONAL-demo'}]
            return 200, json.dumps(result).encode(), {}
        render = m.Render('srv-' + 'a' * 20, 'mock-key')
        record = {}
        with patch.object(m, 'http', side_effect=http), patch.object(m, 'ancestor', return_value=True):
            m.deploy(render, '1.2.3', SHA, CONTRACT, lambda: None, record)
        self.assertEqual(record['status'], 'success')
        self.assertEqual(record['smoke'], 'passed')
        self.assertTrue(any(u == m.ORIGIN + '/demo' for u, _, _, _ in seen))
        self.assertFalse(any('mock-key' in str(h) for u, _, _, h in seen if u.startswith(m.ORIGIN)))
        # A forbidden write accidentally succeeding must fail smoke.
        def writable(url, method='GET', body=None, headers=None):
            if method == 'POST' and url == m.ORIGIN + '/api/v1/accounts':
                return 201, b'{}', {}
            return http(url, method, body, headers)
        with patch.object(m, 'http', side_effect=writable), self.assertRaisesRegex(ValueError, 'read-only'):
            m.smoke(CONTRACT)

    def test_workflow_chain_guards(self):
        # Assert the integration seam, not just the Python implementation.
        root = Path('.github/workflows')
        demo = (root / 'demo-deploy.yml').read_text()
        self.assertNotIn('  push:', demo)
        self.assertNotIn('  release:', demo)
        self.assertIn('  workflow_call:', demo)
        self.assertIn('  workflow_dispatch:', demo)
        self.assertIn('cancel-in-progress: false', demo)
        self.assertIn("github.ref == 'refs/heads/main'", demo)
        self.assertIn('environment: render-demo', demo)
        publish = (root / 'docker-publish.yml').read_text().split('  deploy-demo:')[1]
        self.assertIn('needs: [metadata, publish]', publish)
        self.assertIn("!contains(needs.metadata.outputs.version, '-')", publish)
        self.assertIn("vars.DEMO_AUTO_DEPLOY_ENABLED == 'true'", publish)



if __name__ == '__main__':
    unittest.main()
