import unittest
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


if __name__ == '__main__':
    unittest.main()
