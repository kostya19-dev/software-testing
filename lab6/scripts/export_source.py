import argparse
import errno
import subprocess
import tempfile
from pathlib import Path


def export(revision, source, destination, paths):
    repo = Path(__file__).resolve().parents[2]
    destination = destination.resolve()
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=destination.parent) as temp:
        stage = Path(temp)
        tree = f"{revision}:{source}"
        available = subprocess.run(
            ["git", "-C", repo, "cat-file", "-e", tree], stderr=subprocess.DEVNULL
        )
        if available.returncode:
            repo = stage / "repository.git"
            subprocess.run(["git", "init", "--quiet", "--bare", repo], check=True)
            subprocess.run([
                "git", "-C", repo, "fetch", "--quiet", "--depth=1", "--no-tags",
                "https://github.com/kostya19-dev/software-testing.git", revision,
            ], check=True)
        archive = stage / "source.tar"
        subprocess.run(["git", "-C", repo, "archive", "-o", archive, tree, "--", *paths], check=True)
        files = stage / "files"
        files.mkdir()
        subprocess.run(["tar", "-xf", archive, "-C", files], check=True)
        try:
            files.rename(destination)
        except OSError as exc:
            if exc.errno not in (errno.ENOTEMPTY, errno.EEXIST) or not destination.is_dir():
                raise


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("revision")
    parser.add_argument("source")
    parser.add_argument("destination", type=Path)
    parser.add_argument("paths", nargs="+")
    options = parser.parse_args()
    export(options.revision, options.source, options.destination, options.paths)
