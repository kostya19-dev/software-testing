import copy
import unittest
from report import EXPECTED, validate

def fixture():
    host = {"CpusetCpus":"0","NanoCpus":1000000000,"Memory":536870912,
            "MemorySwap":536870912,"NetworkMode":"none","ReadonlyRootfs":True,
            "PidsLimit":64,"CapDrop":["ALL"],"SecurityOpt":["no-new-privileges"]}
    manifest = {"complete":True,"expected_trials":2,"mode":"smoke","rootfs":{"Layers":["same"]},
                "config":{"cpu":0,"samples":2,"iterations":2,"seed":1},"trials":[]}
    trials = []
    for number in (1,2):
        manifest["trials"].append({"number":number,"image_id":f"image-{number}",
            "container_id":f"container-{number}","host_config":copy.deepcopy(host),
            "rootfs":manifest["rootfs"],"environment":["GOMAXPROCS=1"],"exit_code":0})
        results = []
        for size in (4,64):
            for scenario in ("serialize","deserialize","roundtrip"):
                for codec in (("json","gob") if number==1 else ("gob","json")):
                    r = {"codec":codec,"scenario":scenario,"size":size,"order":len(results),
                         "batch_ns_per_op":100,"total_ns":900,
                         "samples":[{"at_ns":10,"latency_ns":100},{"at_ns":110,"latency_ns":100}]}
                    for phase in ("warmup","batch","requests","teardown"):
                        r[phase] = {"operations":2,"elapsed_ns":200}
                    results.append(r)
        trials.append({"trial":number,"schema":1,"gomaxprocs":1,"seed":1,
                       "go":"go1.27.3","os":"linux","arch":"amd64",
                       "results":results,"elapsed_ns":20000})
    return manifest,trials

class ValidationTests(unittest.TestCase):
    def test_valid_paired_run(self):
        validate(*fixture())

    def test_reused_image_rejected(self):
        m,t = fixture()
        m["trials"][1]["image_id"] = m["trials"][0]["image_id"]
        with self.assertRaisesRegex(ValueError,"distinct"): validate(m,t)

    def test_cpu_difference_rejected(self):
        m,t = fixture()
        m["trials"][1]["host_config"]["CpusetCpus"] = "1"
        with self.assertRaisesRegex(ValueError,"Unequal"): validate(m,t)

    def test_missing_case_rejected(self):
        m,t = fixture()
        t[0]["results"].pop()
        with self.assertRaisesRegex(ValueError,"case"): validate(m,t)

    def test_missing_requests_rejected(self):
        m,t = fixture()
        t[0]["results"][0]["samples"].pop()
        with self.assertRaisesRegex(ValueError,"per-request"): validate(m,t)

    def test_phase_count_corruption_rejected(self):
        m,t = fixture()
        t[0]["results"][0]["teardown"]["operations"] = 0
        with self.assertRaisesRegex(ValueError,"accounting"): validate(m,t)

    def test_order_bias_rejected(self):
        m,t = fixture()
        t[1]["results"][0]["order"],t[1]["results"][1]["order"] = 1,0
        with self.assertRaisesRegex(ValueError,"AB/BA"): validate(m,t)

    def test_corrupt_timing_rejected(self):
        m,t = fixture()
        t[0]["results"][0]["samples"][1]["at_ns"] = 1
        with self.assertRaisesRegex(ValueError,"timing"): validate(m,t)

if __name__=="__main__":
    unittest.main()
