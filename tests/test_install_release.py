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
        for scenario in ('install', 'checksum', 'network', 'archive', 'lock', 'unsupported'):
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as temp:
                root = Path(temp) / 'plugin with spaces'
                (root / 'scripts').mkdir(parents=True)
                (root / 'bin').mkdir()
                shutil.copy(ROOT / 'scripts/install-release.sh', root / 'scripts')
                target = root / 'bin/systemd-lsp'
                target.write_bytes(b'old executable')
                target.chmod(0o755)
                if scenario == 'lock':
                    (root / 'bin/.install-lock').mkdir()
                mocks = Path(temp) / 'mocks'
                mocks.mkdir()
                uname = mocks / 'uname'
                uname.write_text('#!/bin/sh\ncase "$1" in -s) echo Linux;; -m) echo ' +
                                 ('aarch64' if scenario == 'unsupported' else 'x86_64') + ';; esac\n')
                uname.chmod(0o755)
                payload = b'#!/bin/sh\nexit 0\n'
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
[ "$SCENARIO" != network ] || exit 22
case "$url" in
  */latest) : > "$out"; printf '%s' https://github.com/zyosaidouki/systemd-lsp/releases/tag/build-test;;
  */build-test/SHA256SUMS) cp "$FIXTURE_SUM" "$out";;
  */build-test/systemd-lsp-linux-amd64.tar.gz) cp "$FIXTURE_ARCHIVE" "$out";;
  *) exit 23;;
esac
''')
                curl.chmod(0o755)
                env = dict(os.environ, PATH=str(mocks) + os.pathsep + os.environ['PATH'],
                           SCENARIO=scenario, FIXTURE_SUM=str(checksum), FIXTURE_ARCHIVE=str(fixture))
                result = subprocess.run(['sh', str(root / 'scripts/install-release.sh')],
                                        env=env, capture_output=True, timeout=10)
                if scenario == 'install':
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(target.read_bytes(), payload)
                    self.assertTrue(os.access(target, os.X_OK))
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(target.read_bytes(), b'old executable')
                self.assertFalse((root / 'bin/escape').exists())
                self.assertEqual(list((root / 'bin').glob('.install-*')),
                                 [root / 'bin/.install-lock'] if scenario == 'lock' else [])


if __name__ == '__main__':
    unittest.main()
