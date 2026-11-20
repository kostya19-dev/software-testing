import hashlib
from collections import defaultdict
import gzip
import json
import math
from pathlib import Path
import random
import statistics

PERCENTILES = (0.5, 0.75, 0.9, 0.95, 0.99)
SCENARIOS = ("serialize", "deserialize", "roundtrip")
LEGACY = {"write":"serialize", "read":"deserialize"}
EXPECTED = {(c,s,n) for c in ("json","gob") for s in SCENARIOS for n in (4,64)}

def case_keys(manifest):
    return ("compact", "pretty") if manifest.get("profile")=="formats" else (4,64)

def case_label(key):
    return f"{key} JSON · глубина 100" if isinstance(key,str) else f"посещений: {key}"

def scenario_name(name):
    return LEGACY.get(name, name)

def read_summary(directory):
    summary = json.loads((directory/"summary.json").read_text())
    for row in summary["rows"] + summary["comparisons"]:
        row["scenario"] = scenario_name(row["scenario"])
    return summary

def chart_source(directory, scenario, size):
    path = directory/f"{scenario}-{size}.png"
    if path.is_file():
        return path
    for old, new in LEGACY.items():
        if scenario == new:
            return directory/f"{old}-{size}.png"
    return path

PHASES = ("warmup","batch","requests","teardown")

def percentile(values, p):
    """Nearest rank, equal weight per observed request; no interpolation."""
    if not values or not 0 < p <= 1:
        raise ValueError("Empty data or invalid percentile")
    return sorted(values)[math.ceil(p*len(values))-1]

def bootstrap_median(values, seed=20261105):
    rng = random.Random(seed)
    medians = [statistics.median(rng.choices(values,k=len(values))) for _ in range(2000)]
    return [percentile(medians,0.025),percentile(medians,0.975)]

def load(directory):
    directory = Path(directory)
    manifest = json.loads((directory/"manifest.json").read_text())
    for info in manifest.get("inputs", []):
        data = (directory/Path(info["file"]).name).read_bytes()
        if hashlib.sha256(data).hexdigest() != info["sha256"]:
            raise ValueError("Input snapshot checksum mismatch")
    archive = directory/"raw.jsonl.gz"
    if archive.exists():
        with gzip.open(archive,"rt") as stream:
            trials = [json.loads(line) for line in stream]
    else:
        trials = [json.loads(p.read_text()) for p in sorted(directory.glob("trial-*.json"))]
    validate(manifest,trials)
    return manifest,trials

def validate(m, trials):
    if not m["complete"] or len(trials) != m["expected_trials"]:
        raise ValueError("Incomplete experiment")
    if m["mode"] not in ("full","smoke") or (m["mode"]=="full" and len(trials)<100):
        raise ValueError("A full experiment requires at least 100 trials")
    if len(trials)%2:
        raise ValueError("AB/BA order requires an even trial count")
    ids = {t["trial"] for t in trials}
    if ids != set(range(1,len(trials)+1)):
        raise ValueError("Missing/duplicate trial numbers")
    if len(m["trials"]) != len(trials) or {t["number"] for t in m["trials"]} != ids:
        raise ValueError("Missing container provenance")
    for field in ("image_id","container_id"):
        if len({r[field] for r in m["trials"]}) != len(trials):
            raise ValueError("Trials must use distinct images and containers")
    hosts = [r["host_config"] for r in m["trials"]]
    if any(h != hosts[0] for h in hosts):
        raise ValueError("Unequal container configuration")
    cfg = m["config"]
    expected_host = {"CpusetCpus":str(cfg["cpu"]),"NanoCpus":1000000000,
                     "Memory":536870912,"MemorySwap":536870912,
                     "NetworkMode":"none","ReadonlyRootfs":True,"PidsLimit":64,
                     "CapDrop":["ALL"],"SecurityOpt":["no-new-privileges"]}
    if hosts[0] != expected_host:
        raise ValueError("Unexpected CPU/memory/isolation settings")
    for r in m["trials"]:
        if r["exit_code"] != 0 or r["rootfs"] != m["rootfs"]:
            raise ValueError("Failed container or different image contents")
        if r["environment"] != m["trials"][0]["environment"]:
            raise ValueError("Unequal environment")
    versions = {(t["go"],t["os"],t["arch"]) for t in trials}
    if len(versions) != 1:
        raise ValueError("Unequal runtime versions")
    for t in trials:
        if t.get("inputs") != m.get("inputs"):
            raise ValueError("Different input files across trials")
        if t["schema"] != 1 or t["gomaxprocs"] != 1 or t["seed"] != cfg["seed"]:
            raise ValueError("Wrong runtime settings")
        if len(t["results"]) != 12 or {(r["codec"],scenario_name(r["scenario"]),r.get("format",r["size"])) for r in t["results"]} != {(c,s,k) for c in ("json","gob") for s in SCENARIOS for k in case_keys(m)}:
            raise ValueError("Missing/duplicate benchmark case")
        orders = {r["order"] for r in t["results"]}
        if orders != set(range(12)):
            raise ValueError("Invalid execution order")
        pairs = defaultdict(dict)
        for r in t["results"]:
            pairs[(scenario_name(r["scenario"]),r.get("format",r["size"]))][r["codec"]] = r["order"]
            for phase in PHASES:
                want = cfg["samples"] if phase=="requests" else cfg["iterations"]
                if r[phase]["operations"] != want or r[phase]["elapsed_ns"] <= 0:
                    raise ValueError("Invalid phase accounting")
            if len(r["samples"]) != cfg["samples"]:
                raise ValueError("Missing per-request measurements")
            if not math.isclose(r["batch_ns_per_op"],r["batch"]["elapsed_ns"]/cfg["iterations"]):
                raise ValueError("Invalid batch average")
            if r["total_ns"] < sum(r[p]["elapsed_ns"] for p in PHASES):
                raise ValueError("Invalid case duration")
            previous = -1
            for s in r["samples"]:
                if not 0 < s["latency_ns"] or s["at_ns"] < previous or s["at_ns"] > t["elapsed_ns"]:
                    raise ValueError("Invalid timing sample")
                previous = s["at_ns"]
        for order in pairs.values():
            if abs(order["json"]-order["gob"]) != 1 or ((order["json"]<order["gob"]) != (t["trial"]%2==1)):
                raise ValueError("Incorrect paired AB/BA execution")

def summarize(m,trials):
    grouped = defaultdict(list)
    for t in trials:
        for r in t["results"]:
            grouped[(r.get("format",r["size"]),scenario_name(r["scenario"]),r["codec"])].append((t,r))
    rows = []
    for (size,scenario,codec),values in sorted(grouped.items()):
        requests = [s["latency_ns"] for _,r in values for s in r["samples"]]
        batches = [r["batch_ns_per_op"] for _,r in values]
        row = {"size":size,"scenario":scenario,"codec":codec,"trials":len(values),
               "requests":len(requests),"bytes":values[0][1]["bytes"],
               "batch_median_ns":statistics.median(batches),
               "batch_median_ci95_ns":bootstrap_median(batches),
               "request_mean_ns":statistics.mean(requests),
               "request_stddev_ns":statistics.stdev(requests),
               "request_min_ns":min(requests),"request_max_ns":max(requests),
               "outside_100ns_50ms":sum(x<100 or x>50000000 for x in requests),
               "percentiles_ns":{f"p{int(p*100)}":percentile(requests,p) for p in PERCENTILES},
               "phases":{p:{"operations":sum(r[p]["operations"] for _,r in values),
                           "elapsed_ns":sum(r[p]["elapsed_ns"] for _,r in values)} for p in PHASES},
               "total_ns":sum(r["total_ns"] for _,r in values)}
        rows.append(row)
    comparisons = []
    for size in case_keys(m):
        for scenario in SCENARIOS:
            a = {t["trial"]:r["batch_ns_per_op"] for t,r in grouped[(size,scenario,"json")]}
            b = {t["trial"]:r["batch_ns_per_op"] for t,r in grouped[(size,scenario,"gob")]}
            ratios = [b[i]/a[i] for i in sorted(a)]
            comparisons.append({"size":size,"scenario":scenario,"gob_over_json":statistics.median(ratios),
                                "ci95":bootstrap_median(ratios)})
    overheads = [t["overhead"]["batch_ns_per_op"] for t in trials]
    return {"run_id":m["run_id"],"mode":m["mode"],"trials":len(trials),"rows":rows,
            "comparisons":comparisons,"wall_seconds":m["wall_seconds"],
            "benchmark_seconds":sum(t["elapsed_ns"] for t in trials)/1e9,
            "overhead":{"batch_median_ns":statistics.median(overheads),
                        "batch_p99_ns":percentile(overheads,.99),
                        "trials_above_5ns":sum(x>5 for x in overheads),
                        "timer_mean_ns":statistics.mean(t["overhead"]["timer_mean_ns"] for t in trials),
                        "timer_loop_median_ns":statistics.median(t["overhead"]["timer_loop_ns_per_op"] for t in trials),
                        "timer_min_positive_ns":min(t["overhead"]["timer_min_positive_ns"] for t in trials),
                        "timer_zero_samples":sum(t["overhead"]["timer_zero_samples"] for t in trials),
                        "timer_median_ns":statistics.median(t["overhead"]["timer_p50_ns"] for t in trials),
                        "timer_p99_median_ns":statistics.median(t["overhead"]["timer_p99_ns"] for t in trials)},
            "operations":{p:sum(row["phases"][p]["operations"] for row in rows) for p in PHASES},
            "control_operations":sum(t["overhead"]["operations"] for t in trials),
            "validation_codec_calls":24*len(trials)},grouped
