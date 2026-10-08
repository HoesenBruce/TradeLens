// Run with release-please installed outside the repository via NODE_PATH.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const cp = require('node:child_process');
const {Manifest} = require('release-please/build/src/manifest');
const baseline = 'b78c5925778831e7d2d039a4d0139ff1ddfc6e31';
const history = cp.execFileSync('git', ['log', '--format=%H%x09%s', `${baseline}..origin/main`], {encoding:'utf8'}).trim().split('\n').filter(Boolean).map(line => {
  const [sha, message] = line.split('\t');
  return {sha, message, files:['docs/release.md']};
});
async function simulate(message, expected) {
  let crossed = false;
  const github = {
    repository: {owner:'HoesenBruce', repo:'TradeLens'},
    getFileJson: async path => JSON.parse(fs.readFileSync(path, 'utf8')),
    async *releaseIterator() { yield {tagName:'v0.2.1', sha:baseline, notes:''}; },
    async *mergeCommitIterator() {
      if (message) yield {sha:'a'.repeat(40), message, files:['VERSION']};
      yield* history;
      yield {sha:baseline, message:'chore: release 0.2.1', files:['VERSION']};
      crossed = true;
      yield {sha:'b'.repeat(40), message:'feat!: OLD HISTORY MUST BE EXCLUDED', files:['VERSION']};
    },
    async *pullRequestIterator() {},
  };
  const manifest = await Manifest.fromManifest(github, 'main');
  const prs = await manifest.buildPullRequests();
  assert.equal(crossed, false, 'must stop at the actual released SHA');
  assert.equal(prs.length, expected ? 1 : 0);
  if (!expected) return;
  const pr = prs[0];
  assert.equal(pr.version.toString(), expected);
  const updates = Object.fromEntries(pr.updates.map(u => [u.path, u.updater.updateContent(fs.readFileSync(u.path,'utf8'))]));
  assert.equal(updates.VERSION.trim(), expected);
  assert.equal(JSON.parse(updates['.release-please-manifest.json'])['.'], expected);
  assert.equal(JSON.parse(updates['web/package.json']).version, expected);
  assert.equal(JSON.parse(updates['mobile/package.json']).version, expected);
  assert.equal(JSON.parse(updates['mobile/app.json']).expo.version, expected);
  assert(updates['CHANGELOG.md'].includes('## ['+expected+']'));
  assert(updates['CHANGELOG.md'].includes('## [0.13.0]'), 'preserve upstream history');
  assert(!updates['CHANGELOG.md'].includes('OLD HISTORY MUST BE EXCLUDED'));
  assert(updates['CHANGELOG.md'].indexOf('## ['+expected+']') < updates['CHANGELOG.md'].indexOf('## [0.13.0]'));
  console.log(`PASS ${message}: ${expected}; source boundary and all version files verified`);
}
(async () => {
  await simulate('', null);
  await simulate('fix: repair TradeLens Release Please automation', '0.2.2');
  await simulate('feat: next feature', '0.3.0');
  await simulate('feat!: incompatible API', '1.0.0');
})().catch(error => {console.error(error); process.exitCode=1;});
