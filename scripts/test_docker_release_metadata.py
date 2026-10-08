import unittest
import io
import json
import os
import runpy
import subprocess
import tempfile
from unittest.mock import patch
import urllib.error
from importlib.util import spec_from_file_location, module_from_spec
from pathlib import Path

spec = spec_from_file_location('metadata', Path(__file__).with_name('docker-release-metadata.py'))
module = module_from_spec(spec)
spec.loader.exec_module(module)


class Tags(unittest.TestCase):
    def test_policy(self):
        sha = 'a' * 40
        release = {'tag_name': 'v1.2.3', 'draft': False, 'prerelease': False}
        self.assertEqual(module.metadata('', sha)['tags'], f'sha-{sha}')
        self.assertEqual(module.metadata('1.2.3', sha, sha, release, release)['tags'], f'sha-{sha},1.2.3,1.2,1,latest')
        pre = dict(release, tag_name='v1.2.3-rc.1', prerelease=True)
        self.assertEqual(module.metadata('1.2.3-rc.1', sha, sha, pre)['tags'], f'sha-{sha},1.2.3-rc.1')
        for version, tag_sha, rel, latest in [
            ('1.2.3', 'b' * 40, release, release),
            ('1.2.3', sha, release, {'tag_name': 'v2.0.0'}),
            ('1.2.3', sha, dict(release, draft=True), release),
            ('1.2.3', sha, dict(release, prerelease=True), release),
            ('1.2.3', sha, None, None),
            ('01.2.3', sha, release, release),
            ('1.2.3-rc.01', sha, pre, None),
            ('1.2.3\nlatest', sha, release, release),
        ]:
            with self.subTest(version=version, release=rel):
                with self.assertRaises(ValueError):
                    module.metadata(version, sha, tag_sha, rel, latest)


    def test_reusable_source_resolution(self):
        release = {'tag_name': 'v0.2.1', 'draft': False, 'prerelease': False}
        original = subprocess.check_output
        def read_only_api(command, **kwargs):
            if command[0] == 'gh':
                return json.dumps(release).encode()
            return original(command, **kwargs)
        with tempfile.NamedTemporaryFile() as output:
            env = dict(GITHUB_EVENT_NAME='push', GITHUB_REF='refs/heads/main',
                       GITHUB_REPOSITORY='HoesenBruce/TradeLens', INPUT_VERSION='0.2.1',
                       GITHUB_OUTPUT=output.name)
            with patch.dict(os.environ, env), patch('subprocess.check_output', read_only_api):
                runpy.run_path(str(Path(__file__).with_name('docker-release-metadata.py')), run_name='__main__')
            result = Path(output.name).read_text()
            self.assertIn('commit=b78c5925778831e7d2d039a4d0139ff1ddfc6e31', result)
            self.assertIn('version=0.2.1', result)
            with patch.dict(os.environ, dict(env, GITHUB_EVENT_NAME='workflow_dispatch')):
                with patch('subprocess.check_output', read_only_api), self.assertRaises(ValueError):
                    runpy.run_path(str(Path(__file__).with_name('docker-release-metadata.py')), run_name='__main__')
            with patch.dict(os.environ, dict(env, GITHUB_EVENT_NAME='workflow_dispatch', GITHUB_REF='refs/tags/v0.2.1')):
                with patch('subprocess.check_output', read_only_api):
                    runpy.run_path(str(Path(__file__).with_name('docker-release-metadata.py')), run_name='__main__')
            with patch.dict(os.environ, dict(env, EXPECTED_SOURCE_SHA='a'*40)):
                with patch('subprocess.check_output', read_only_api), self.assertRaises(ValueError):
                    runpy.run_path(str(Path(__file__).with_name('docker-release-metadata.py')), run_name='__main__')

    def test_registry_fails_closed(self):
        def absent(request):
            if isinstance(request, str):
                return io.StringIO('{"token":"public-read"}')
            raise urllib.error.HTTPError(request.full_url, 404, 'missing', {}, None)
        module.check_registry('tradelens-api', ['0.2.2'], absent)
        def existing(request):
            return io.StringIO('{"token":"public-read"}')
        with self.assertRaises(ValueError):
            module.check_registry('tradelens-api', ['0.2.2'], existing)
        def denied(request):
            if isinstance(request, str):
                return io.StringIO('{"token":"public-read"}')
            raise urllib.error.HTTPError(request.full_url, 403, 'denied', {}, None)
        with self.assertRaises(urllib.error.HTTPError):
            module.check_registry('tradelens-api', ['0.2.2'], denied)


if __name__ == '__main__':
    unittest.main()
