import copy
import unittest
from report import percentile, bootstrap_median, validate

class StatisticsTests(unittest.TestCase):
    def test_nearest_rank(self):
        values = list(range(1,101))
        for p,want in ((.5,50),(.75,75),(.9,90),(.95,95),(.99,99)):
            self.assertEqual(percentile(values,p),want)
        self.assertEqual(percentile([9],.99),9)
        self.assertEqual(percentile([3,1,2,4],.5),2)

    def test_rejects_invalid_percentiles(self):
        for values,p in (([],.5),([1],0),([1],1.1)):
            with self.assertRaises(ValueError): percentile(values,p)

    def test_bootstrap_constant(self):
        self.assertEqual(bootstrap_median([7]*100),[7,7])

    def test_rejects_incomplete_runs(self):
        with self.assertRaisesRegex(ValueError,"Incomplete"):
            validate({"complete":False},[])
        with self.assertRaisesRegex(ValueError,"at least 100"):
            validate({"complete":True,"expected_trials":2,"mode":"full"},[{},{}])

if __name__=="__main__":
    unittest.main()
