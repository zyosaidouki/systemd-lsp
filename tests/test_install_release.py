"""Exercise the installer without network or changes to an installed plugin."""
import hashlib
import io
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class InstallerTest(unittest.TestCase):
    def test_install_and_failures(self):
        for scenario in ('install', 'fresh', 'update', 'dev', 'already', 'identical',
                         'checksum', 'network', 'archive', 'lock', 'unsupported'):
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as temp:
                root = Path(temp) / 'plugin with spaces'
                (root / 'scripts').mkdir(parents=True)
                (root / 'bin').mkdir()
                shutil.copy(ROOT / 'scripts/install-release.sh', root / 'scripts')
                target = root / 'bin/systemd-lsp'
                target.write_bytes(b'old executable')
                target.chmod(0o755)
                if scenario in ('update', 'dev', 'already'):
                    current = {'update': 'build-old', 'dev': 'dev', 'already': 'build-test'}[scenario]
                    target.write_text('#!/bin/sh\necho "systemd-lsp ' + current + '"\n')
                if scenario == 'fresh':
                    target.unlink()
                if scenario == 'lock':
                    (root / 'bin/.install-lock').mkdir()
                mocks = Path(temp) / 'mocks'
                mocks.mkdir()
                uname = mocks / 'uname'
                uname.write_text('#!/bin/sh\ncase "$1" in -s) echo Linux;; -m) echo ' +
                                 ('aarch64' if scenario == 'unsupported' else 'x86_64') + ';; esac\n')
                uname.chmod(0o755)
                payload = b'#!/bin/sh\ncase "$1" in --version) echo "systemd-lsp build-test";; esac\n'
                if scenario == 'identical':
                    # An older binary that does not implement --version.
                    payload = b'#!/bin/sh\ncase "$1" in --version) exit 1;; esac\n'
                    target.write_bytes(payload)
                original = target.read_bytes() if target.exists() else None
                before = target.stat() if target.exists() else None
                buf = io.BytesIO()
                with tarfile.open(fileobj=buf, mode='w:gz') as archive:
                    entry = tarfile.TarInfo('../escape' if scenario == 'archive' else 'systemd-lsp')
                    entry.size = len(payload)
                    entry.mode = 0o755
                    archive.addfile(entry, io.BytesIO(payload))
                fixture = Path(temp) / 'archive.tar.gz'
                fixture.write_bytes(buf.getvalue())
                digest = hashlib.sha256(buf.getvalue()).hexdigest()
                if scenario == 'checksum':
                    digest = '0' * 64
                checksum = Path(temp) / 'SHA256SUMS'
                checksum.write_text(digest + '  systemd-lsp-linux-amd64.tar.gz\n')
                curl = mocks / 'curl'
                curl.write_text('''#!/bin/sh
set -eu
out=''
while [ "$#" -gt 0 ]; do
  case "$1" in -o) out=$2; shift;; esac
  url=$1
  shift
done
printf '%s\n' "$url" >> "$REQUEST_LOG"
[ "$SCENARIO" != network ] || exit 22
case "$url" in
  */latest) : > "$out"; printf '%s' https://github.com/zyosaidouki/systemd-lsp/releases/tag/build-test;;
  */build-test/SHA256SUMS) cp "$FIXTURE_SUM" "$out";;
  */build-test/systemd-lsp-linux-amd64.tar.gz) cp "$FIXTURE_ARCHIVE" "$out";;
  *) exit 23;;
esac
''')
                curl.chmod(0o755)
                request_log = Path(temp) / 'requests'
                env = dict(os.environ, PATH=str(mocks) + os.pathsep + os.environ['PATH'],
                           SCENARIO=scenario, FIXTURE_SUM=str(checksum), FIXTURE_ARCHIVE=str(fixture),
                           REQUEST_LOG=str(request_log))
                result = subprocess.run(['sh', str(root / 'scripts/install-release.sh')],
                                        env=env, capture_output=True, timeout=10)
                if scenario in ('install', 'fresh', 'update', 'dev'):
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(target.read_bytes(), payload)
                    self.assertTrue(os.access(target, os.X_OK))
                    previous = {'install': 'unknown', 'fresh': 'not-installed', 'update': 'build-old', 'dev': 'dev'}[scenario]
                    self.assertEqual(result.stdout.decode().splitlines()[-1],
                                     f'systemd-lsp updated: {previous} -> build-test')
                    # A second update must only check latest, preserving the file.
                    before = target.stat()
                    request_log.write_text('')
                    repeated = subprocess.run(['sh', str(root / 'scripts/install-release.sh')],
                                              env=env, capture_output=True, timeout=10)
                    self.assertEqual(repeated.returncode, 0, repeated.stderr)
                    self.assertEqual(repeated.stdout.decode().strip(), 'systemd-lsp already: build-test')
                    self.assertEqual(target.stat().st_ino, before.st_ino)
                    self.assertEqual(request_log.read_text().splitlines(),
                                     ['https://github.com/zyosaidouki/systemd-lsp/releases/latest'])
                elif scenario in ('already', 'identical'):
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(result.stdout.decode().splitlines()[-1], 'systemd-lsp already: build-test')
                    self.assertEqual(target.read_bytes(), original)
                    self.assertEqual(target.stat().st_ino, before.st_ino)
                    if scenario == 'already':
                        self.assertEqual(request_log.read_text().splitlines(),
                                         ['https://github.com/zyosaidouki/systemd-lsp/releases/latest'])
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(target.read_bytes(), original)
                self.assertFalse((root / 'bin/escape').exists())
                self.assertEqual(list((root / 'bin').glob('.install-*')),
                                 [root / 'bin/.install-lock'] if scenario == 'lock' else [])


if __name__ == '__main__':
    unittest.main()
