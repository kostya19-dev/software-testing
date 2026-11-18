#!/usr/bin/env python3
"""Export a complete full run; refuses to overwrite existing evidence."""
import argparse
import fcntl
import tempfile
import gzip
import hashlib
import json
from pathlib import Path
import shutil
from report import load, summarize, read_summary, render

def archive(source, target):
    m,trials = load(source)
    if m["mode"] != "full":
        raise ValueError("Only a complete full experiment can be published")
    if target.exists() and any(target.iterdir()):
        raise ValueError("Destination is not empty; choose a new directory")
    target.mkdir(parents=True,exist_ok=True)
    s,_ = summarize(m,trials)
    if read_summary(source) != s:
        raise ValueError("Report does not match raw measurements")
    for info in m.get("inputs", []):
        name = Path(info["file"]).name
        shutil.copy2(source/name,target/name)
    shutil.copy2(source/"manifest.json", target/"manifest.json")
    if (source/"benchmark-source.json.gz").is_file():
        shutil.copy2(source/"benchmark-source.json.gz",target/"benchmark-source.json.gz")
    render(m,trials,target)
    with gzip.GzipFile(filename=str(target/"raw.jsonl.gz"),mode="wb",mtime=0) as f:
        for t in trials:
            f.write((json.dumps(t,ensure_ascii=False,separators=(",",":"))+"\n").encode())
    hashes = [f"{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}" for p in sorted(target.iterdir()) if p.is_file()]
    (target/"SHA256SUMS").write_text("\n".join(hashes)+"\n")
    print(target)

def publish_evidence(source, destination):
    from verify import verify
    results = destination.parent/"results"
    results.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="evidence-update-", dir=results) as tmp:
        stage = Path(tmp)/"new"
        archive(source, stage)
        verify(stage)
        with (results/".evidence.lock").open("a") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            backup = results/("previous-" + Path(tmp).name)
            if destination.is_symlink():
                raise ValueError("Expected a local evidence directory")
            if destination.exists():
                destination.rename(backup)
            try:
                stage.rename(destination)
            except BaseException:
                if backup.exists():
                    backup.rename(destination)
                raise
    print(f"EVIDENCE={destination}",flush=True)


if __name__ == "__main__":
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("source",type=Path)
    p.add_argument("target",type=Path)
    a = p.parse_args()
    archive(a.source,a.target)
