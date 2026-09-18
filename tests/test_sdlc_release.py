import importlib.machinery
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

loader = importlib.machinery.SourceFileLoader("release", str(Path(__file__).resolve().parents[1] / "bin/sdlc-release"))
spec = importlib.util.spec_from_loader(loader.name, loader)
r = importlib.util.module_from_spec(spec)
loader.exec_module(r)


class GateTests(unittest.TestCase):
    def setUp(self):
        self.sha = "a" * 40
        self.artifact = "sha256:" + "b" * 64
        self.config = {"acceptance": ["bin/accept"], "delivery_profile": "service"}
        self.impl = [{"number": 2, "head": "c" * 40, "merge": self.sha, "author": "Builder"}]
        self.candidate = {"repo": "o/r", "issue": 1, "sha": self.sha, "artifact": self.artifact,
                          "implementation": self.impl, "spec_sha256": r.e.sha256(b"Specification"),
                          "config_sha256": r.e.sha256(r.e.encoded(self.config))}

    def gate(self, actor="Reviewer", subject=None, candidate=None):
        with patch.object(r.e, "identity"), patch.object(r, "on_default_branch"), patch.object(r, "latest_candidate", return_value=candidate or self.candidate), \
             patch.object(r.e, "api", return_value=subject or {"state": "open", "body": "Specification"}), \
             patch.object(r, "remote_config", return_value=self.config), \
             patch.object(r, "implementation", return_value=self.impl), \
             patch.object(r.e, "trusted_receipt", return_value="https://evidence") as verify:
            result = r.gate("o/r", 1, self.sha, self.artifact, actor)
            self.assertEqual(verify.call_args.kwargs["candidate_sha256"], r.e.sha256(r.e.encoded(self.candidate)))
            return result

    def test_gate_binds_acceptance_to_nomination_and_implementation(self):
        self.assertEqual(self.gate()["candidate"], self.candidate)

    def test_changed_spec_closed_issue_wrong_artifact_and_self_promotion_fail(self):
        cases = [{"subject": {"state": "open", "body": "Changed"}},
                 {"subject": {"state": "closed", "body": "Specification"}},
                 {"candidate": dict(self.candidate, artifact="sha256:" + "d" * 64)},
                 {"actor": "Builder"}]
        for case in cases:
            with self.subTest(case=case), self.assertRaises(r.e.Failure):
                self.gate(**case)

    def test_unmerged_or_missing_implementation_is_rejected(self):
        with self.assertRaises(r.e.Failure):
            r.implementation("o/r", self.sha, [])
        with patch.object(r.e, "api", return_value={"merged": False}):
            with self.assertRaises(r.e.Failure):
                r.implementation("o/r", self.sha, [2])

    def test_merged_but_unreviewed_implementation_is_rejected(self):
        pr = {"merged": True, "base": {"repo": {"full_name": "o/r"}},
              "merge_commit_sha": self.sha, "head": {"sha": self.sha}, "user": {"login": "Builder"}}
        with patch.object(r.e, "api", side_effect=[pr, {"status": "identical"}]), patch.object(r.e, "pages", return_value=[]):
            with self.assertRaisesRegex(r.e.Failure, "approval"):
                r.implementation("o/r", self.sha, [2])

    def test_candidate_outside_default_branch_is_rejected(self):
        with patch.object(r.e, "api", side_effect=[{"default_branch": "main"}, {"status": "diverged"}]):
            with self.assertRaises(r.e.Failure):
                r.on_default_branch("o/r", self.sha)


class PromotionTests(unittest.TestCase):
    def test_resume_verifies_without_repeating_promotion(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder) / "repo"
            root.mkdir()
            output = Path(folder) / "receipt"
            output.mkdir()
            candidate = {"sha": "a" * 40, "artifact": "sha256:" + "b" * 64}
            config = {"promotion": ["echo promotion >> deployed"],
                      "release_verification": ["echo verified"], "rollback": ["echo restore"]}
            receipt = {"candidate": candidate, "state": "promotion", "commands": []}
            (output / "release.json").write_text(json.dumps(receipt))
            checked = {"candidate": candidate, "config": config, "acceptance": "https://evidence"}
            with patch.object(r, "gate", return_value=checked), \
                 patch.object(r.e, "clean_head", return_value="a" * 40), \
                 patch.object(r.e, "config_at", return_value=config):
                with self.assertRaises(r.e.Failure):
                    r.release(root, "o/r", 1, "a" * 40, candidate["artifact"], "Reviewer", output)
                result = r.release(root, "o/r", 1, "a" * 40, candidate["artifact"], "Reviewer", output, True)
            self.assertEqual(result["state"], "verified")
            self.assertFalse((root / "deployed").exists())
            self.assertEqual([c["phase"] for c in result["commands"]], ["release_verification"])

    def test_missing_promotion_capability_is_not_success(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder) / "repo"
            root.mkdir()
            config = {"promotion": [], "release_verification": ["true"], "rollback": ["true"]}
            with patch.object(r, "gate", return_value={"candidate": {}, "config": config}), \
                 patch.object(r.e, "clean_head", return_value="a" * 40), patch.object(r.e, "config_at", return_value=config):
                with self.assertRaises(r.e.Failure):
                    r.release(root, "o/r", 1, "a" * 40, "sha256:" + "b" * 64, "Reviewer", Path(folder) / "output")


if __name__ == "__main__":
    unittest.main()
