#!/usr/bin/env python3
import os
import shutil
import urllib.request
from pathlib import Path
from urllib.parse import quote
from urllib.parse import urlsplit, unquote


crashdir = Path(os.environ.get("CRASHDIR", "/etc/ShellCrash"))
tmpdir = Path(os.environ.get("TMPDIR", "/tmp/ShellCrash"))
cfg_path = crashdir / "configs" / "ShellCrash.cfg"
providers_cfg = crashdir / "configs" / "providers.cfg"
providers_uri_cfg = crashdir / "configs" / "providers_uri.cfg"
servers_list = crashdir / "configs" / "servers.list"
core_config = crashdir / "yamls" / "config.yaml"
core_config_new = tmpdir / "clash_config.yaml"

(crashdir / "configs").mkdir(parents=True, exist_ok=True)
(crashdir / "yamls").mkdir(parents=True, exist_ok=True)
tmpdir.mkdir(parents=True, exist_ok=True)
cfg_path.touch(exist_ok=True)


def read_cfg():
    data = {}
    for line in cfg_path.read_text(encoding="utf-8").splitlines():
        if "=" in line and not line.startswith("#"):
            k, v = line.split("=", 1)
            data[k.strip()] = v.strip()
    return data


def write_cfg(data):
    lines = [f"{k}={v}" for k, v in data.items()]
    cfg_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def strip_quotes(v):
    return v.strip().strip("'").strip('"')


cfg = read_cfg()

# 1) 默认 core=meta（Mihomo），只走 clash/yaml 主路径
cfg["crashcore"] = "meta"

# 2) Subconverter 选项1语义：聚合 providers + uri 到 Url
provider_urls = []
if providers_cfg.exists():
    for line in providers_cfg.read_text(encoding="utf-8").splitlines():
        parts = line.split()
        if len(parts) >= 2 and not parts[1].startswith("./providers/"):
            provider_urls.append(parts[1])

uri_urls = []
if providers_uri_cfg.exists():
    for line in providers_uri_cfg.read_text(encoding="utf-8").splitlines():
        s = line.strip()
        if not s or s.startswith("#"):
            continue
        parts = s.split()
        if len(parts) >= 2:
            name = parts[0]
            link = parts[1]
            if name == "vmess":
                uri_urls.append(link)
            else:
                uri_urls.append(f"{link}#{name}")

url_value = "|".join(provider_urls + uri_urls).strip("|")
if not url_value:
    raise SystemExit("no providers found")

cfg["Url"] = f"'{url_value}'"
cfg["Https"] = ""

# 3) 读取默认索引并拼接 subconvert 链接
server_link = int(strip_quotes(cfg.get("server_link", "1")) or "1")
rule_link = int(strip_quotes(cfg.get("rule_link", "1")) or "1")
exclude = strip_quotes(cfg.get("exclude", ""))
include = strip_quotes(cfg.get("include", ""))

user_agent = strip_quotes(cfg.get("user_agent", "auto"))
if not user_agent or user_agent == "auto":
    user_agent = "clash.meta/mihomo"
if user_agent == "none":
    user_agent = ""

server_rows = []
rule_rows = []
for line in servers_list.read_text(encoding="utf-8").splitlines():
    s = line.strip()
    if not s or s.startswith("#"):
        continue
    parts = s.split()
    if len(parts) < 3:
        continue
    if parts[0].startswith(("3", "4")):
        server_rows.append(parts)
    if parts[0].startswith("5"):
        rule_rows.append(parts)

server = server_rows[server_link - 1][2]
server_ua = server_rows[server_link - 1][3] if len(server_rows[server_link - 1]) > 3 else "ua"
config_tpl = rule_rows[rule_link - 1][2]

https_url = (
    f"{server}/sub?target=clash&{server_ua}={quote(user_agent)}&insert=true&new_name=true&scv=true&udp=true"
    f"&exclude={quote(exclude)}&include={quote(include)}&url={quote(url_value)}&config={quote(config_tpl)}"
)

cfg["server_link"] = str(server_link)
cfg["rule_link"] = str(rule_link)
cfg["user_agent"] = user_agent
cfg["Https"] = f"'{https_url}'"
write_cfg(cfg)

# 4) 下载在线生成配置
if https_url.startswith("file://"):
    local_path = unquote(urlsplit(https_url).path)
    content = Path(local_path).read_text(encoding="utf-8", errors="ignore")
else:
    with urllib.request.urlopen(https_url, timeout=20) as r:
        content = r.read().decode("utf-8", errors="ignore")
core_config_new.write_text(content, encoding="utf-8")

# 5) 极简校验：必须含配置关键字，且含 hy2 特征
low = content.lower()
if "proxies:" not in low and "proxy-providers:" not in low and "server:" not in low:
    raise SystemExit("invalid yaml config")
if "hysteria2" not in low and "type: hysteria" not in low:
    raise SystemExit("hy2 not found")

# 6) 覆盖正式配置（不同才备份）
if core_config.exists():
    old = core_config.read_text(encoding="utf-8", errors="ignore")
    if old != content:
        shutil.copy2(core_config, core_config.with_suffix(".yaml.bak"))
        shutil.move(str(core_config_new), str(core_config))
    else:
        core_config_new.unlink(missing_ok=True)
else:
    shutil.move(str(core_config_new), str(core_config))

print(f"ok: generated {core_config}")
