"""Fail closed before emitting registry tags. Uses only Python's standard library."""
import datetime
import json
import os
from pathlib import Path
import re
import subprocess


def metadata(version, sha, tag_sha=None, release=None, latest=None):
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        raise ValueError("Expected full source SHA")
    tags = [f"sha-{sha}"]
    if version:
        match = re.fullmatch(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?", version)
        if not match or len(version) > 128:
            raise ValueError("Invalid container release version")
        pre = match[4] is not None
        if pre and any(p.isdigit() and len(p) > 1 and p.startswith('0') for p in match[4][1:].split('.')):
            raise ValueError("Invalid numeric prerelease identifier")
        if tag_sha != sha or not release or release['draft'] or release['tag_name'] != f'v{version}' or release['prerelease'] != pre:
            raise ValueError("Published release, tag SHA and prerelease flag must match")
        tags.append(version)
        if not pre:
            if not latest or latest['tag_name'] != f'v{version}':
                raise ValueError("Only the latest stable GitHub Release can emit moving tags")
            tags += [f'{match[1]}.{match[2]}', match[1], 'latest']
    return {'version': version or 'dev', 'commit': sha, 'tags': ','.join(tags)}


if __name__ == '__main__':
    version = os.environ.get('INPUT_VERSION', '')
    if os.environ['GITHUB_EVENT_NAME'] == 'release':
        event = json.loads(Path(os.environ['GITHUB_EVENT_PATH']).read_text())
        tag = event['release']['tag_name']
        if not tag.startswith('v'):
            raise ValueError('Release tag must start with v')
        version = tag[1:]
    sha = subprocess.check_output(['git', 'rev-parse', 'HEAD'], text=True).strip()
    tag_sha = release = latest = None
    if version:
        # Validate before putting the version in a command argument/API path.
        if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?', version):
            raise ValueError('Invalid version')
        repo = os.environ['GITHUB_REPOSITORY']
        tag_sha = subprocess.check_output(['git', 'rev-parse', f'refs/tags/v{version}^{{commit}}'], text=True).strip()
        release = json.loads(subprocess.check_output(['gh', 'api', f'repos/{repo}/releases/tags/v{version}']))
        if not release['prerelease']:
            latest = json.loads(subprocess.check_output(['gh', 'api', f'repos/{repo}/releases/latest']))
    result = metadata(version, sha, tag_sha, release, latest)
    result['build_time'] = datetime.datetime.now(datetime.timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
    with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
        for key, value in result.items():
            if key == 'tags':
                output.write('tags<<TAGS\n')
                for tag in value.split(','):
                    output.write(f'type=raw,value={tag}\n')
                output.write('TAGS\n')
            else:
                output.write(f'{key}={value}\n')
