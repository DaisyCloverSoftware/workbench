"""Synthetic subprocess fixtures, not live GitHub or Unreal proof."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / "ops" / "github-runner-routing.sh"
EMPTY = {"total_count": 0, "runners": []}
WINDOWS = {"id": 7, "os": "windows", "status": "online", "busy": False,
           "labels": ["self-hosted", "Windows", "X64"]}


class RunnerRoutingTests(unittest.TestCase):
    def run_script(self, repo=EMPTY, org=EMPTY, args=("ExampleOrg/game",)):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "fixture.json").write_text(json.dumps({"repository": repo, "organization": org}))
            fake = root / "gh"
            fake.write_text("#!" + sys.executable + '''
import json, os, sys
from pathlib import Path
assert "GH_DEBUG" not in os.environ
assert os.environ.get("GH_PROMPT_DISABLED") == "1"
assert sys.argv[1:6] == ["api", "--hostname", "github.com", "--method", "GET"]
assert len(sys.argv) == 9 and sys.argv[7] == "--jq"
endpoint = sys.argv[6]
assert endpoint in ("repos/ExampleOrg/game/actions/runners?per_page=100&page=1", "orgs/ExampleOrg/actions/runners?per_page=100&page=1")
root = Path(os.environ["FIXTURE_ROOT"])
with (root / "calls").open("a") as stream: stream.write(endpoint + "\\n")
value = json.loads((root / "fixture.json").read_text())["repository" if endpoint.startswith("repos/") else "organization"]
if isinstance(value, str):
    print(value, file=sys.stderr)
    sys.exit(1)
print(json.dumps(value))
''')
            fake.chmod(0o700)
            env = {**os.environ, "PATH": str(root) + os.pathsep + os.environ["PATH"],
                   "FIXTURE_ROOT": str(root), "GH_DEBUG": "api"}
            result = subprocess.run(["bash", str(SCRIPT), *args], env=env,
                                    capture_output=True, text=True, timeout=10)
            calls = (root / "calls").read_text().splitlines() if (root / "calls").exists() else []
            return result.returncode, json.loads(result.stdout), result.stderr, calls

    def test_empty_is_complete(self):
        code, data, _, calls = self.run_script()
        self.assertEqual(code, 0)
        self.assertEqual(len(calls), 2)
        self.assertEqual(data["routing_conclusion"], "no_matching_registered_runner_visible")
        self.assertEqual(data["native_unreal_proof"], "not_obtained")

    def test_windows_candidate_is_not_proof(self):
        code, data, _, _ = self.run_script(org={"total_count": 1, "runners": [WINDOWS]})
        self.assertEqual(code, 0)
        self.assertEqual(data["scopes"]["organization"]["online_idle_matching_ids"], [7])
        self.assertEqual(data["routing_conclusion"], "matching_runner_visible_eligibility_not_proven")

    def test_offline_is_not_idle_capacity(self):
        _, data, _, _ = self.run_script(org={"total_count": 1, "runners": [{**WINDOWS, "status": "offline"}]})
        self.assertEqual(data["scopes"]["organization"]["online_idle_matching_ids"], [])

    def test_linux_cannot_be_relabelled_into_windows(self):
        _, data, _, _ = self.run_script(org={"total_count": 1, "runners": [{**WINDOWS, "os": "linux"}]})
        self.assertEqual(data["routing_conclusion"], "no_matching_registered_runner_visible")

    def test_permission_failure_is_unknown_and_redacted(self):
        code, data, stderr, calls = self.run_script(repo="HTTP 403 sensitive-error-must-not-leak")
        self.assertEqual(code, 1)
        self.assertEqual(len(calls), 2)
        self.assertEqual(data["routing_conclusion"], "incomplete_inventory")
        self.assertEqual(data["scopes"]["repository"]["error_code"], "github_http_403")
        self.assertNotIn("sensitive-error", json.dumps(data) + stderr)

    def test_partial_is_unknown(self):
        code, data, _, _ = self.run_script(org={"total_count": 101, "runners": []})
        self.assertEqual(code, 1)
        self.assertEqual(data["routing_conclusion"], "incomplete_inventory")

    def test_boolean_total_is_rejected(self):
        code, data, _, _ = self.run_script(org={"total_count": False, "runners": []})
        self.assertEqual(code, 1)
        self.assertEqual(data["scopes"]["organization"]["error_code"], "invalid_runner_response")

    def test_duplicate_runner_ids_are_rejected(self):
        code, _, _, _ = self.run_script(org={"total_count": 2, "runners": [WINDOWS, WINDOWS]})
        self.assertEqual(code, 1)

    def test_invalid_shape_is_rejected(self):
        code, _, _, _ = self.run_script(org=None)
        self.assertEqual(code, 1)

    def test_input_cannot_select_arbitrary_endpoint_or_shell(self):
        for args in [(), ("ExampleOrg/game", "extra"), ("https://github.com/ExampleOrg/game",),
                     ("ExampleOrg/game;echo",), ("ExampleOrg/game/actions/runners",), ("../game",)]:
            with self.subTest(args=args):
                code, _, _, calls = self.run_script(args=args)
                self.assertEqual(code, 2)
                self.assertEqual(calls, [])

    def test_busy_runner_is_not_idle_capacity(self):
        _, data, _, _ = self.run_script(org={"total_count": 1, "runners": [{**WINDOWS, "busy": True}]})
        self.assertEqual(data["scopes"]["organization"]["online_idle_matching_ids"], [])


if __name__ == "__main__":
    unittest.main(verbosity=2)
