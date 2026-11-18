import json
from pathlib import Path

DEPTH = 100
VISITS = 4
ROOT = Path(__file__).resolve().parents[1]


def build():
    referral = None
    for level in range(DEPTH, 1, -1):  # 100, 99, ..., 2
        node = {"Level": level, "Doctor": f"doctor-{level:03}"}
        if referral is not None:
            node["Next"] = referral
        referral = node
    return {
        "ID": 42,
        "Name": "Учебный пациент",
        "Email": "patient@example.invalid",
        "Active": True,
        "Visits": [{"Code": f"procedure-{i:03}", "Duration": 15 + i % 5, "Price": 1200 + 17 * i}
                   for i in range(VISITS)],
        "Clinic": {"Name": "Учебная клиника",
                   "Department": {"Name": "Терапия",
                                  "Team": {"ID": 7, "Doctor": {"ID": 101, "Name": "Врач"}}}},
        "Referral": referral,
    }


def depth(v):
    if isinstance(v, dict):
        return 1 + max((depth(x) for x in v.values()), default=0)
    if isinstance(v, list):
        return 1 + max((depth(x) for x in v), default=0)
    return 0


if __name__ == "__main__":
    data = build()
    assert depth(data) == DEPTH, depth(data)
    out = ROOT / "bench" / "inputs"
    compact = json.dumps(data, ensure_ascii=False, separators=(",", ":"))
    pretty = json.dumps(data, ensure_ascii=False, indent=2)
    (out / "compact-100.json").write_text(compact, encoding="utf-8", newline="\n")
    (out / "pretty-100.json").write_text(pretty + "\n", encoding="utf-8", newline="\n")
    print(f"depth={depth(data)} compact={len(compact.encode())}B pretty={len(pretty.encode())}B")
