import shutil
import subprocess
import tempfile
from pathlib import Path

root = Path(__file__).resolve().parents[1]
project = root.parent / "project" / "VitaLineClinic"
out = root / "reports" / "coverage"
out.mkdir(parents=True, exist_ok=True)
packages = {
    "domain": "internal/domain",
    "usecase": "internal/usecase",
    "storage/memory": "internal/storage/memory",
    "storage/mongodb": "internal/storage/mongodb",
}
with tempfile.TemporaryDirectory(prefix="tiopo-branches-") as tmp:
    stage = Path(tmp) / "module"
    stage.mkdir()
    for name in ("go.mod", "go.sum"):
        shutil.copy2(root / name, stage / name)
    text = (
        (stage / "go.mod")
        .read_text()
        .replace("module vitaline-clinic/lab1", "module vitaline-clinic")
    )
    text = (
        "\n".join(
            line
            for line in text.splitlines()
            if "replace vitaline-clinic =>" not in line
            and "vitaline-clinic v0.0.0" not in line
        )
        + "\n"
    )
    (stage / "go.mod").write_text(text)
    for file in (project / "internal").rglob("*.go"):
        if file.name.endswith("_test.go"):
            continue
        dest = stage / file.relative_to(project)
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(file, dest)
    shutil.copytree(root / "testutil", stage / "internal" / "testutil")
    for src, dst in packages.items():
        for file in (root / src).glob("*_test.go"):
            test = file.read_text().replace(
                "vitaline-clinic/lab1/testutil",
                "vitaline-clinic/internal/testutil",
            )
            (stage / dst / ("lab1_" + file.name)).write_text(test)
    summaries = []
    for src, dst in packages.items():
        run = subprocess.run(
            [str(root / ".tools/gobco"), "-branch", "-test=-count=1", "./" + dst],
            cwd=stage,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
        )
        (out / ("branches-" + src.replace("/", "-") + ".txt")).write_text(run.stdout)
        if run.returncode:
            raise SystemExit(run.stdout)
        summary = next(
            (
                line
                for line in run.stdout.splitlines()
                if line.startswith("Branch coverage:")
            ),
            None,
        )
        if summary is None:
            raise SystemExit("Missing branch coverage result: " + run.stdout)
        summaries.append(src + ": " + summary)
    (out / "branches.txt").write_text("\n".join(summaries) + "\n")
    print("\n".join(summaries))
