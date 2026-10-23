import json
from pathlib import Path
import tempfile
import unittest

from report import trend_history


class TrendHistoryTests(unittest.TestCase):
    def test_finished_history_excludes_concurrent_and_future_runs(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            def run(name, start, finish=None):
                folder = root / name
                folder.mkdir()
                state = dict(id=name, started=start, finished=finish)
                (folder / "summary.json").write_text(json.dumps(state))
                (folder / "junit.xml").write_text(
                    '<testsuite><testcase/><testcase><failure/></testcase>'
                    '<testcase><error/></testcase><testcase><skipped/></testcase></testsuite>'
                )
                return state

            run("old", "2026-10-22T09:00:00+00:00", "2026-10-22T09:01:00+00:00")
            run("active", "2026-10-22T09:02:00+00:00")
            run("overlap", "2026-10-22T09:03:00+00:00", "2026-10-22T10:01:00+00:00")
            current = run("current", "2026-10-22T10:00:00+00:00", "2026-10-22T10:02:00+00:00")
            run("future", "2026-10-22T11:00:00+00:00", "2026-10-22T11:01:00+00:00")
            order, history = trend_history(root / "current", current, root)
            self.assertEqual(order, 4)
            self.assertEqual(len(history), 1)
            self.assertEqual(history[0]["reportName"], "ЛР2 — old")
            self.assertEqual(history[0]["data"], dict(passed=1, failed=1, broken=1, skipped=1, unknown=0, total=4))

    def test_history_is_newest_first_and_limited(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for n in range(25):
                folder = root / str(n)
                folder.mkdir()
                stamp = f"2026-10-{n + 1:02d}T00:00:00+00:00"
                (folder / "summary.json").write_text(json.dumps(dict(id=str(n), started=stamp, finished=stamp)))
            state = dict(id="new", started="2026-10-30T00:00:00+00:00")
            order, history = trend_history(root / "new", state, root)
            self.assertEqual(order, 26)
            self.assertEqual(len(history), 19)
            self.assertEqual(history[0]["buildOrder"], 25)
            self.assertEqual(history[-1]["buildOrder"], 7)


if __name__ == "__main__":
    unittest.main()
