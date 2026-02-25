import os
import subprocess
import tempfile
from pathlib import Path


def test_default_meta_subconvert_option1_hy2_flow():
    repo_root = Path(__file__).resolve().parents[1]
    script = repo_root / "scripts" / "minimal_flow_default_meta_hy2_subconvert.py"

    with tempfile.TemporaryDirectory() as td:
        tdp = Path(td)
        crashdir = tdp / "ShellCrash"
        (crashdir / "configs").mkdir(parents=True)
        (crashdir / "yamls").mkdir(parents=True)
        (crashdir / "jsons").mkdir(parents=True)

        (crashdir / "configs" / "providers.cfg").write_text(
            "sub1 https://sub.example.com/a.yaml 3 12 clash.meta ##\n",
            encoding="utf-8",
        )
        (crashdir / "configs" / "providers_uri.cfg").write_text(
            "hy2a hy2://node-a\nhy2b hy2://node-b\n",
            encoding="utf-8",
        )

        sub_path = tdp / "sub"
        sub_path.write_text(
            "proxies:\n"
            "  - {name: hy2-a, type: hysteria2, server: a.example.com, port: 443}\n"
            "  - {name: hy2-b, type: hysteria2, server: b.example.com, port: 443}\n"
            "rules:\n"
            "  - MATCH,DIRECT\n",
            encoding="utf-8",
        )
        (crashdir / "configs" / "servers.list").write_text(
            f"401 test file://{tdp} ua\n"
            "501 rule https://rules.example.com/template.ini\n",
            encoding="utf-8",
        )

        env = os.environ.copy()
        env["CRASHDIR"] = str(crashdir)
        env["TMPDIR"] = str(tdp / "tmp")
        proc = subprocess.run(
            ["python3", str(script)],
            cwd=str(repo_root),
            env=env,
            capture_output=True,
            text=True,
        )

        assert proc.returncode == 0, proc.stderr + proc.stdout
        cfg_text = (crashdir / "configs" / "ShellCrash.cfg").read_text(encoding="utf-8")
        out_yaml = (crashdir / "yamls" / "config.yaml").read_text(encoding="utf-8")

        assert "crashcore=meta" in cfg_text
        assert f"Https='file://{tdp}/sub?" in cfg_text
        assert "type: hysteria2" in out_yaml
