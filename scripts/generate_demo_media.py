#!/usr/bin/env python3
"""Generate portfolio-ready demo screenshots and a short MP4 for crypto-profiler."""

from __future__ import annotations

import html
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MEDIA_ROOT = ROOT / "docs" / "media"
SCREENSHOT_DIR = MEDIA_ROOT / "screenshots"
VIDEO_DIR = MEDIA_ROOT / "video"
SAMPLE_REPORTS = ROOT / "docs" / "sample-reports"
CHROME = Path("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
WIDTH = 1920
HEIGHT = 1080


SCENES = [
    {
        "slug": "01-demo-overview",
        "type": "card",
        "title": "Crypto Profiler Demo Overview",
        "subtitle": "Portfolio-grade multi-chain crypto risk profiling for AML, sanctions, fraud, and regtech-style wallet review.",
        "bullets": [
            "Shared validator surface with analyst-facing --report output",
            "Trace-aware Ethereum plus curated Solana, Bitcoin, and ERC-20 Layer 1 cases",
            "Explainable scoring with visible reasons, counterparties, and review framing",
        ],
        "footer": "Recommended order: Ethereum -> Solana -> Bitcoin -> ERC-20",
        "duration": 6,
        "output": SCREENSHOT_DIR / "01-demo-overview.png",
    },
    {
        "slug": "02-ethereum-curated-report",
        "type": "report",
        "title": "Ethereum Curated Analyst Report",
        "subtitle": "Trace-aware Ethereum Layer 1 case with high-risk entity context and internal call breadth.",
        "command": "go run ./cmd/validator --report --dataset ./data/cases/curated-enriched/tornado-router-high-risk.json",
        "report_file": SAMPLE_REPORTS / "ethereum-tornado-router.txt",
        "notes": [
            "Shows the strongest end-to-end path in the repo today",
            "Pairs label-based risk with trace-aware context",
            "Demonstrates the report mode clearly in one screen",
        ],
        "duration": 14,
        "output": SCREENSHOT_DIR / "02-ethereum-curated-report.png",
    },
    {
        "slug": "03-solana-curated-report",
        "type": "report",
        "title": "Solana Curated Analyst Report",
        "subtitle": "Stablecoin-flow and authority-role analysis rendered in the same validator/report surface.",
        "command": "go run ./cmd/validator --report --dataset ./data/cases/curated-solana/solana-stablecoin-authority-operator.json",
        "report_file": SAMPLE_REPORTS / "solana-authority-operator.txt",
        "notes": [
            "Makes the chain-specific Solana modeling visible",
            "Highlights dominant role, mint, and repeated operational linkage",
            "Useful for showing the repo is not Ethereum-only",
        ],
        "duration": 14,
        "output": SCREENSHOT_DIR / "03-solana-curated-report.png",
    },
    {
        "slug": "04-bitcoin-curated-report",
        "type": "report",
        "title": "Bitcoin Curated Analyst Report",
        "subtitle": "UTXO-flow language, spend-heavy routing behavior, and reviewable risk framing.",
        "command": "go run ./cmd/validator --report --dataset ./data/cases/curated-bitcoin/bitcoin-broad-spend-heavy-operational-hub.json",
        "report_file": SAMPLE_REPORTS / "bitcoin-operational-hub.txt",
        "notes": [
            "Uses Bitcoin-native receipts/spends wording",
            "Shows repeated interaction concentration in a non-EVM context",
            "Keeps the score nuanced instead of overclaiming malice",
        ],
        "duration": 14,
        "output": SCREENSHOT_DIR / "04-bitcoin-curated-report.png",
    },
    {
        "slug": "05-erc20-curated-report",
        "type": "report",
        "title": "ERC-20 Curated Analyst Report",
        "subtitle": "Token-surface breadth, trusted protocol context, and repeated counterparty patterns.",
        "command": "go run ./cmd/validator --report --dataset ./data/cases/curated-erc20/erc20-uniswap-v2-router-trusted-token-hub.json",
        "report_file": SAMPLE_REPORTS / "erc20-uniswap-v2-router.txt",
        "notes": [
            "Shows the Wave 2 ERC-20 dataset-mode work clearly",
            "Balances broad token activity with trusted-service context",
            "Demonstrates contextual scoring instead of blanket high-risk labeling",
        ],
        "duration": 14,
        "output": SCREENSHOT_DIR / "05-erc20-curated-report.png",
    },
]


VIDEO_CLOSING = {
    "slug": "06-closing-card",
    "type": "card",
    "title": "Crypto Profiler",
    "subtitle": "Explainable multi-chain wallet profiling with curated cases, validator dataset mode, and analyst-facing report output.",
    "bullets": [
        "Use docs/DEMO-WALKTHROUGH.md for the live talk track",
        "Use docs/sample-reports/README.md for static review assets",
        "Use the same validator commands shown in this video for demos",
    ],
    "footer": "README -> ARCHITECTURE -> DEMO-WALKTHROUGH -> sample reports",
    "duration": 6,
}


BASE_CSS = """
* { box-sizing: border-box; }
html, body {
  margin: 0;
  width: 100%;
  height: 100%;
  background:
    radial-gradient(circle at top left, #1d4ed8 0%, rgba(29, 78, 216, 0.12) 24%, transparent 42%),
    radial-gradient(circle at bottom right, #0f766e 0%, rgba(15, 118, 110, 0.12) 22%, transparent 40%),
    linear-gradient(180deg, #07111f 0%, #0b1220 58%, #070c17 100%);
  color: #ecf3ff;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}
body {
  padding: 58px 72px;
}
.frame {
  width: 100%;
  height: 100%;
  border-radius: 28px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(7, 15, 30, 0.72);
  box-shadow: 0 28px 80px rgba(0, 0, 0, 0.34);
  backdrop-filter: blur(20px);
  overflow: hidden;
}
.card {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 64px 72px;
}
.eyebrow {
  color: #8fb6ff;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}
.title {
  margin: 14px 0 0;
  font-size: 52px;
  line-height: 1.08;
}
.subtitle {
  max-width: 1100px;
  margin-top: 18px;
  color: #c5d5f4;
  font-size: 24px;
  line-height: 1.5;
}
.bullets {
  margin: 34px 0 0;
  padding: 0;
  list-style: none;
}
.bullets li {
  margin-top: 18px;
  padding-left: 28px;
  position: relative;
  color: #eef4ff;
  font-size: 26px;
  line-height: 1.45;
}
.bullets li:before {
  content: "";
  position: absolute;
  left: 0;
  top: 14px;
  width: 11px;
  height: 11px;
  border-radius: 999px;
  background: linear-gradient(180deg, #7dd3fc 0%, #60a5fa 100%);
}
.footer {
  color: #8ea4c6;
  font-size: 18px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}
.report-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 380px;
  gap: 28px;
  padding: 42px;
}
.report-main {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.report-header {
  padding: 10px 8px 0;
}
.report-title {
  margin: 6px 0 0;
  font-size: 42px;
  line-height: 1.1;
}
.report-subtitle {
  margin-top: 12px;
  color: #bfd0ee;
  font-size: 22px;
  line-height: 1.45;
}
.command {
  padding: 18px 22px;
  border-radius: 18px;
  background: rgba(8, 16, 33, 0.9);
  border: 1px solid rgba(125, 211, 252, 0.2);
  color: #9ae6b4;
  font-family: "Menlo", "SFMono-Regular", monospace;
  font-size: 18px;
  white-space: pre-wrap;
  line-height: 1.45;
}
.terminal {
  border-radius: 24px;
  background: linear-gradient(180deg, #111827 0%, #09111f 100%);
  border: 1px solid rgba(148, 163, 184, 0.16);
  overflow: hidden;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}
.terminal-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 20px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
  background: rgba(255, 255, 255, 0.03);
}
.dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
}
.dot.red { background: #fb7185; }
.dot.amber { background: #fbbf24; }
.dot.green { background: #34d399; }
.bar-label {
  margin-left: 10px;
  color: #8ea4c6;
  font-size: 14px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.terminal pre {
  margin: 0;
  padding: 24px 26px 26px;
  color: #ebf3ff;
  font-family: "Menlo", "SFMono-Regular", monospace;
  font-size: 18px;
  line-height: 1.36;
  white-space: pre-wrap;
}
.report-side {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.note-box {
  border-radius: 22px;
  padding: 22px 22px 24px;
  background: rgba(8, 16, 33, 0.84);
  border: 1px solid rgba(148, 163, 184, 0.14);
}
.note-box h3 {
  margin: 0;
  color: #8fb6ff;
  font-size: 16px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}
.note-box ul {
  margin: 14px 0 0;
  padding-left: 18px;
}
.note-box li {
  margin-top: 10px;
  color: #e7efff;
  font-size: 18px;
  line-height: 1.45;
}
"""


def ensure_dirs() -> None:
    SCREENSHOT_DIR.mkdir(parents=True, exist_ok=True)
    VIDEO_DIR.mkdir(parents=True, exist_ok=True)


def make_card_html(scene: dict[str, object]) -> str:
    bullets = "\n".join(
        f"<li>{html.escape(str(item))}</li>" for item in scene["bullets"]
    )
    return f"""<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <style>{BASE_CSS}</style>
  </head>
  <body>
    <div class="frame card">
      <div>
        <div class="eyebrow">Crypto Profiler</div>
        <h1 class="title">{html.escape(str(scene["title"]))}</h1>
        <div class="subtitle">{html.escape(str(scene["subtitle"]))}</div>
        <ul class="bullets">{bullets}</ul>
      </div>
      <div class="footer">{html.escape(str(scene["footer"]))}</div>
    </div>
  </body>
</html>
"""


def make_report_html(scene: dict[str, object]) -> str:
    report_text = Path(scene["report_file"]).read_text(encoding="utf-8").strip()
    notes = "\n".join(f"<li>{html.escape(str(item))}</li>" for item in scene["notes"])
    return f"""<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <style>{BASE_CSS}</style>
  </head>
  <body>
    <div class="frame report-layout">
      <div class="report-main">
        <div class="report-header">
          <div class="eyebrow">Crypto Profiler</div>
          <h1 class="report-title">{html.escape(str(scene["title"]))}</h1>
          <div class="report-subtitle">{html.escape(str(scene["subtitle"]))}</div>
        </div>
        <div class="command">$ {html.escape(str(scene["command"]))}</div>
        <div class="terminal">
          <div class="terminal-bar">
            <div class="dot red"></div>
            <div class="dot amber"></div>
            <div class="dot green"></div>
            <div class="bar-label">Analyst report output</div>
          </div>
          <pre>{html.escape(report_text)}</pre>
        </div>
      </div>
      <div class="report-side">
        <div class="note-box">
          <h3>What This Shows</h3>
          <ul>{notes}</ul>
        </div>
        <div class="note-box">
          <h3>Portfolio Value</h3>
          <ul>
            <li>Readable enough for interviews, recruiter review, and GitHub skimming</li>
            <li>Demonstrates one shared validator surface across multiple chain models</li>
            <li>Uses real curated cases instead of mock data or placeholder screenshots</li>
          </ul>
        </div>
      </div>
    </div>
  </body>
</html>
"""


def chrome_render(html_path: Path, image_path: Path, chrome_workdir: Path) -> None:
    if not CHROME.exists():
        raise SystemExit(f"Chrome binary not found at {CHROME}")

    if image_path.exists():
        image_path.unlink()

    chrome_cmd = [
        str(CHROME),
        "--headless=new",
        "--disable-gpu",
        "--hide-scrollbars",
        "--disable-background-networking",
        "--no-first-run",
        "--no-default-browser-check",
        "--allow-file-access-from-files",
        f"--user-data-dir={chrome_workdir}",
        f"--window-size={WIDTH},{HEIGHT}",
        f"--screenshot={image_path}",
        html_path.resolve().as_uri(),
    ]

    proc = subprocess.Popen(
        chrome_cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )
    try:
        for _ in range(80):
            if image_path.exists() and image_path.stat().st_size > 0:
                break
            proc.poll()
            if proc.returncode is not None and not image_path.exists():
                output = proc.stdout.read() if proc.stdout else ""
                raise RuntimeError(
                    f"Chrome render failed for {html_path.name}:\n{output}"
                )
            proc.wait(timeout=0.25)
    except subprocess.TimeoutExpired:
        pass
    finally:
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=2)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=2)

    if not image_path.exists() or image_path.stat().st_size == 0:
        output = proc.stdout.read() if proc.stdout else ""
        raise RuntimeError(f"Expected screenshot not created for {html_path.name}:\n{output}")


def render_screenshot(scene: dict[str, object], build_dir: Path) -> None:
    html_path = build_dir / f"{scene['slug']}.html"
    output_path = Path(scene["output"])
    html_text = make_card_html(scene) if scene["type"] == "card" else make_report_html(scene)
    html_path.write_text(html_text, encoding="utf-8")
    chrome_dir = build_dir / f"{scene['slug']}-chrome"
    chrome_dir.mkdir(parents=True, exist_ok=True)
    chrome_render(html_path, output_path, chrome_dir)


def render_closing_card(build_dir: Path) -> Path:
    html_path = build_dir / f"{VIDEO_CLOSING['slug']}.html"
    png_path = build_dir / f"{VIDEO_CLOSING['slug']}.png"
    html_path.write_text(make_card_html(VIDEO_CLOSING), encoding="utf-8")
    chrome_dir = build_dir / f"{VIDEO_CLOSING['slug']}-chrome"
    chrome_dir.mkdir(parents=True, exist_ok=True)
    chrome_render(html_path, png_path, chrome_dir)
    return png_path


def build_video(scene_images: list[Path], closing_image: Path, build_dir: Path) -> Path:
    manifest = build_dir / "video_manifest.txt"
    lines: list[str] = []
    for scene, image in zip(SCENES, scene_images):
        lines.append(f"file '{image.resolve()}'")
        lines.append(f"duration {scene['duration']}")
    lines.append(f"file '{closing_image.resolve()}'")
    lines.append(f"duration {VIDEO_CLOSING['duration']}")
    lines.append(f"file '{closing_image.resolve()}'")
    manifest.write_text("\n".join(lines) + "\n", encoding="utf-8")

    video_path = VIDEO_DIR / "crypto-profiler-demo.mp4"
    ffmpeg_cmd = [
        "ffmpeg",
        "-y",
        "-f",
        "concat",
        "-safe",
        "0",
        "-i",
        str(manifest),
        "-vf",
        "fps=30,format=yuv420p",
        "-c:v",
        "libx264",
        "-pix_fmt",
        "yuv420p",
        "-movflags",
        "+faststart",
        str(video_path),
    ]
    subprocess.run(ffmpeg_cmd, check=True)
    return video_path


def main() -> int:
    ensure_dirs()
    build_dir = Path(tempfile.mkdtemp(prefix="crypto-profiler-demo-media-"))
    try:
        rendered_images: list[Path] = []
        for scene in SCENES:
            render_screenshot(scene, build_dir)
            rendered_images.append(Path(scene["output"]))
        closing_image = render_closing_card(build_dir)
        video_path = build_video(rendered_images, closing_image, build_dir)
        print("Generated screenshots:")
        for image in rendered_images:
            print(image)
        print("Generated video:")
        print(video_path)
        return 0
    finally:
        shutil.rmtree(build_dir, ignore_errors=True)


if __name__ == "__main__":
    sys.exit(main())
