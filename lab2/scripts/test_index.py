import json
from pathlib import Path
import tempfile
import unittest

from pipeline import render_index


class ReportIndexTests(unittest.TestCase):
    def test_latest_five_are_visible_and_older_runs_are_collapsed(self):
        with tempfile.TemporaryDirectory() as directory:
            lab = Path(directory)
            (lab / ".runs").mkdir()
            reports = lab / "reports"
            reports.mkdir()
            for n in range(1, 6):
                name = f"parallel-20261023-101815-25944-{n}"
                folder = reports / name
                folder.mkdir()
                (folder / "summary.json").write_text(json.dumps(dict(id=name, started=f"2026-10-23T10:18:1{n}+00:00", finished=True, stages=dict(unit="passed", integration="passed", e2e="passed"))))
            folder = reports / "stand-off-old"
            folder.mkdir()
            (folder / "summary.json").write_text(json.dumps(dict(id=folder.name, started="2026-10-23T09:00:00+00:00", finished=True, stages=dict(unit="skipped"), preparation_error="Стенд недоступен: MongoDB. Тесты не восстанавливают сервисы.")))
            render_index(lab)
            page = (reports / "index.html").read_text()
            visible, archived = page.split("<details>")
            self.assertEqual(visible.count("<li>"), 5)
            self.assertNotIn("stand-off-old", visible)
            self.assertIn("stand-off-old", archived)
            self.assertNotIn("Тесты не восстанавливают сервисы", page)
            self.assertNotIn("Каждый запуск имеет отдельный отчёт", page)


if __name__ == "__main__":
    unittest.main()
