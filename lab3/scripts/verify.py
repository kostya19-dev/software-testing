#!/usr/bin/env python3
"""Check evidence integrity, experiment invariants and reproducible statistics."""
import argparse
import gzip
import hashlib
import json
from pathlib import Path
from report import load, summarize, read_summary

def verify(directory):
    directory = directory.resolve()
    sums = directory/"SHA256SUMS"
    if sums.exists():
        listed = set()
        for line in sums.read_text().splitlines():
            digest,name = line.split("  ",1)
            path = (directory/name).resolve()
            if path.parent != directory or name in listed:
                raise ValueError("Invalid checksum entry")
            if hashlib.sha256(path.read_bytes()).hexdigest() != digest:
                raise ValueError(f"Checksum mismatch: {name}")
            listed.add(name)
        actual = {p.name for p in directory.iterdir() if p.is_file() and p.name!="SHA256SUMS"}
        if listed != actual:
            raise ValueError("Checksum inventory does not cover all artifacts")
    m,trials = load(directory)
    expected,_ = summarize(m,trials)
    if read_summary(directory) != expected:
        raise ValueError("Published summary differs from raw measurements")
    source_hash = hashlib.sha256()
    root = Path(__file__).resolve().parents[1]
    for path in sorted((root/"bench").glob("*.go")):
        source_hash.update(path.name.encode())
        source_hash.update(path.read_bytes())
    for info in m.get("inputs", []):
        fixture = root/"bench"/"inputs"/Path(info["file"]).name
        if hashlib.sha256(fixture.read_bytes()).hexdigest() != info["sha256"]:
            raise ValueError(f"Current input differs from the measured version: {fixture.name}")
    if source_hash.hexdigest() != m["source_sha256"]:
        snapshot = directory/"benchmark-source.json.gz"
        if not sums.is_file() or not snapshot.is_file():
            raise ValueError("Current benchmark source differs from the measured version")
        measured = json.loads(gzip.decompress(snapshot.read_bytes()))
        archived_hash = hashlib.sha256()
        for name, content in sorted(measured.items()):
            archived_hash.update(name.encode())
            archived_hash.update(content.encode())
        if archived_hash.hexdigest() != m["source_sha256"]:
            raise ValueError("Archived benchmark source differs from the measured version")
        print("Checking the measured source snapshot; the workspace now contains a newer runner.")
    print(f"PASS: {len(trials)} trials, {len(expected['rows'])} cases, "
          f"{expected['operations']['requests']} individual requests; source, provenance and statistics verified")

if __name__ == "__main__":
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("directory",type=Path)
    verify(p.parse_args().directory)
