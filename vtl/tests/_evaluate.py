import boto3
import glob
import json
import os.path
import yaml
import sys
import unittest

# todo: flags (path, only certain file, etc.)

appsync = boto3.client('appsync')

class GeneratedError(Exception):
    pass

class AppsyncTemplateEvaluation(unittest.TestCase):
    def __init__(self, testname: str, test: dict, vtlDir: str, debug: bool):
        super(AppsyncTemplateEvaluation, self).__init__("test_evaluate")
        self.testname = testname
        self.test = test
        self.vtlDir = vtlDir
        self.debug = debug

    def setUp(self):
        print()
        print(self.testname)

    def test_evaluate(self):
        context = self.test["context"]
        file = os.path.join(self.vtlDir, self.test["file"])

        if "error" in self.test and self.test["error"] == True:
            self.assertRaises(GeneratedError, evaluate_template, context, file)
            if self.debug:
                print(f"Generated Error")
        else:
            result = evaluate_template(context, file)
            if self.debug:
                print(json.dumps(result, indent=2))
            self.assertEqual(result, self.test["expect"])


def evaluate_template(ctx: dict, filename: str):
    with open(filename) as f:
        template = f.read()

    evaluation = appsync.evaluate_mapping_template(
        template = template,
        context  = json.dumps(ctx),
    )

    if "error" in evaluation and "message" in evaluation["error"] and evaluation["error"]["message"] != "":
        raise GeneratedError(evaluation["error"]["message"])

    if evaluation["evaluationResult"].strip() == "":
        return None

    result = json.loads(evaluation["evaluationResult"])
    return result

def run(casesDir: str, vtlDir: str, only: list, debug: bool = False):
    suite = unittest.TestSuite()

    for filename in glob.glob(os.path.join(casesDir, "*.yaml")):
        basename = os.path.splitext(os.path.basename(filename))[0]
        if len(only) > 0 and basename not in only:
            continue

        with open(filename, "r") as stream:
            testsDef = yaml.safe_load(stream)

            for test in testsDef["tests"]:
                testname = f"{basename}/{test['name']}"
                suite.addTest(AppsyncTemplateEvaluation(testname=testname, test=test, vtlDir=vtlDir, debug=debug))

    unittest.TextTestRunner().run(suite)

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--vtl', type=str, nargs='?', const=1, default="..")
    parser.add_argument('--cases', type=str, nargs='?', const=1, default="cases")
    parser.add_argument('--only', type=str, nargs='+', default=[])
    parser.add_argument('--debug', action='store_true')
    args = parser.parse_args()
    run(casesDir=args.cases, vtlDir=args.vtl, only=args.only, debug=args.debug)