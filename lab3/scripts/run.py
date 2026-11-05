#!/usr/bin/env python3
"""Sequential isolated trials: unique image, fresh container, fixed CPU/config."""
import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import platform
import subprocess
import sys
import time
import uuid

ROOT = Path(__file__).resolve().parents[1]

def command(args, timeout=180):
    p = subprocess.run(args, cwd=ROOT, text=True, capture_output=True, timeout=timeout)
    if p.returncode:
        raise RuntimeError(f"{' '.join(args)}\n{p.stderr}\n{p.stdout}")
    return p.stdout.strip()

def save(path, data):
    temporary = path.with_suffix(".tmp")
    temporary.write_text(json.dumps(data, ensure_ascii=False, indent=2)+"\n")
    temporary.replace(path)

def run(args):
    if args.trials < (2 if args.smoke else 100) or args.trials % 2:
        raise ValueError("Need an even number of trials, >=100 (>=2 for --smoke)")
    if args.iterations < 1 or args.samples < 1 or args.cpu < 0:
        raise ValueError("Invalid operations/sample/CPU count")
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    run_id = stamp.lower()+"-"+uuid.uuid4().hex[:8]
    out = ROOT / "results" / run_id
    out.mkdir(parents=True, exist_ok=False)
    binary = ROOT / ".build/bench"
    if not binary.is_file():
        raise ValueError("Run make build first")
    source_hash = hashlib.sha256()
    for path in sorted((ROOT / "bench").glob("*.go")):
        source_hash.update(path.name.encode())
        source_hash.update(path.read_bytes())
    config = {"cpu":args.cpu, "cpus":1, "memory":"512m", "network":"none",
              "read_only":True, "pids_limit":64, "iterations":args.iterations,
              "samples":args.samples, "seed":20261105}
    manifest = {"schema":1, "run_id":run_id, "mode":"smoke" if args.smoke else "full",
        "expected_trials":args.trials, "config":config, "complete":False,
        "started_utc":stamp, "host":platform.platform(),
        "binary_sha256":hashlib.sha256(binary.read_bytes()).hexdigest(),
        "source_sha256":source_hash.hexdigest(), "trials":[]}
    started = time.monotonic()
    base_tag = "tiopo-lab3-base:"+run_id
    current_tag = None
    current_container = None
    try:
        manifest["docker"] = json.loads(command(["docker","info","--format","{{json .}}"]))
        # Keep only non-sensitive machine details.
        manifest["docker"] = {k:manifest["docker"].get(k) for k in
                              ("ServerVersion","NCPU","MemTotal","Architecture","OSType","KernelVersion")}
        command(["docker","build","--network=none","-t",base_tag,"."])
        base = json.loads(command(["docker","image","inspect",base_tag]))[0]
        manifest["base_image"] = base["Id"]
        manifest["rootfs"] = base["RootFS"]
        manifest["binary_build"] = command(["go","version","-m",str(binary)])
        save(out/"manifest.json",manifest)
        for number in range(1,args.trials+1):
            current_tag = f"tiopo-lab3-trial:{run_id}-{number:03}"
            command(["docker","build","--network=none","-f","Dockerfile.trial",
                     "--build-arg","BASE="+base_tag,"--build-arg",f"TRIAL={run_id}-{number}",
                     "-t",current_tag,"."])
            image = json.loads(command(["docker","image","inspect",current_tag]))[0]
            if image["RootFS"] != base["RootFS"] or image["Config"] != {
                **base["Config"], "Labels":{**(base["Config"].get("Labels") or {}),
                                          "tiopo.lab3.trial":f"{run_id}-{number}"}}:
                raise RuntimeError("Trial image differs beyond its identification label")
            current_container = command(["docker","create","--name",f"lab3-{run_id}-{number:03}",
                "--network=none","--read-only","--cap-drop=ALL","--security-opt=no-new-privileges",
                "--cpuset-cpus",str(args.cpu),"--cpus=1","--memory=512m","--memory-swap=512m",
                "--pids-limit=64",current_tag,"-trial",str(number),
                "-iterations",str(args.iterations),"-samples",str(args.samples),"-seed","20261105"])
            before = json.loads(command(["docker","inspect",current_container]))[0]
            trial_start = time.monotonic()
            raw = command(["docker","start","--attach",current_container],timeout=120)
            after = json.loads(command(["docker","inspect",current_container]))[0]
            if after["State"]["ExitCode"] != 0 or after["State"]["OOMKilled"]:
                raise RuntimeError(f"Trial {number} failed: {after['State']}")
            trial = json.loads(raw)
            if trial["trial"] != number or trial["gomaxprocs"] != 1:
                raise RuntimeError("Unexpected trial identity/configuration")
            save(out/f"trial-{number:03}.json",trial)
            hc = before["HostConfig"]
            manifest["trials"].append({"number":number,"image_id":image["Id"],
                "image_tag":current_tag,"container_id":current_container,
                "started_at":after["State"]["StartedAt"],"finished_at":after["State"]["FinishedAt"],
                "exit_code":after["State"]["ExitCode"],"wall_seconds":time.monotonic()-trial_start,
                "rootfs":image["RootFS"],"environment":before["Config"]["Env"],
                "host_config":{k:hc[k] for k in ("CpusetCpus","NanoCpus","Memory","MemorySwap",
                    "NetworkMode","ReadonlyRootfs","PidsLimit","CapDrop","SecurityOpt")}})
            save(out/"manifest.json",manifest)
            command(["docker","rm",current_container]); current_container = None
            command(["docker","image","rm",current_tag]); current_tag = None
            print(f"{number:3}/{args.trials}: {trial['elapsed_ns']/1e9:.3f}s, {len(trial['results'])} cases",flush=True)
        manifest["complete"] = True
    except BaseException as exc:
        manifest["error"] = str(exc)
        raise
    finally:
        for target,verb in ((current_container,["rm","-f"]),(current_tag,["image","rm"]),(base_tag,["image","rm"])):
            if target:
                try: command(["docker",*verb,target])
                except Exception as exc: print(f"Cleanup: {exc}",file=sys.stderr)
        manifest["wall_seconds"] = time.monotonic()-started
        save(out/"manifest.json",manifest)
        print(f"RESULT={out}",flush=True)
    command([sys.executable,"scripts/report.py",str(out)],timeout=180)
    print(f"REPORT={out/'report.html'}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--trials",type=int,default=100)
    parser.add_argument("--cpu",type=int,default=0)
    parser.add_argument("--iterations",type=int,default=256)
    parser.add_argument("--samples",type=int,default=256)
    parser.add_argument("--smoke",action="store_true")
    try: run(parser.parse_args())
    except (ValueError,RuntimeError,subprocess.TimeoutExpired) as exc:
        print(exc,file=sys.stderr); sys.exit(1)
