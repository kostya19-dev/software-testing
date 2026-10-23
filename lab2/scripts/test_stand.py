from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import pipeline
import stand


class ExistingStandTests(unittest.TestCase):
    def test_stopped_mongo_is_rejected_without_starting_it(self):
        response = subprocess.CompletedProcess([], 0, json.dumps([
            {"Name": "/tiopo-lab2-mongo-1", "State": {"Running": False}}
        ]), "")
        with patch.object(stand.subprocess, "run", return_value=response) as run:
            with self.assertRaisesRegex(RuntimeError, "Стенд недоступен"):
                stand.check()
        self.assertEqual(run.call_count, 1)
        self.assertEqual(run.call_args.args[0][:2], ["docker", "inspect"])

    def test_failed_preparation_records_error_without_starting_services(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            lab = root / "lab2"
            lab.mkdir()
            with patch.object(pipeline, "LAB", lab), patch.object(pipeline, "ROOT", root), \
                 patch.object(stand, "check", side_effect=RuntimeError("MongoDB stopped")), \
                 patch.object(stand, "up") as up, patch.object(pipeline.Run, "render"):
                run = pipeline.Run("unavailable")
                with self.assertRaisesRegex(RuntimeError, "MongoDB stopped"):
                    run.prepare()
                self.assertTrue(run.state["cleaned"])
                self.assertEqual(run.state["preparation_error"], "MongoDB stopped")
                self.assertTrue((run.report / "prepare-junit.xml").exists())
                up.assert_not_called()

    def test_five_parallel_leases_are_isolated_and_sixth_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(stand, "LAB", Path(directory)):
            with ThreadPoolExecutor(max_workers=5) as pool:
                slots = list(pool.map(stand.lease, [f"run-{i}" for i in range(5)]))
            self.assertEqual(set(slots), set(range(1, 6)))
            self.assertEqual(len({stand.slot_database(slot) for slot in slots}), 5)
            with self.assertRaisesRegex(RuntimeError, "заняты"):
                stand.lease("sixth")
            stand.lease("run-2", release=True)
            self.assertEqual(stand.lease("sixth"), slots[2])
            owners = json.loads((Path(directory) / ".runs/slots.json").read_text())
            self.assertEqual(len(owners), 5)


if __name__ == "__main__":
    unittest.main()
