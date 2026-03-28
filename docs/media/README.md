# Demo Media Assets

This directory contains portfolio-ready demo media generated from the curated report examples already checked into the repo.

## Included Assets

### Screenshots

- `screenshots/01-demo-overview.png`
- `screenshots/02-ethereum-curated-report.png`
- `screenshots/03-solana-curated-report.png`
- `screenshots/04-bitcoin-curated-report.png`
- `screenshots/05-erc20-curated-report.png`

What they demonstrate:

- overview framing for recruiter or hiring-manager review
- Ethereum trace-aware analyst reporting
- Solana stablecoin authority-role reporting
- Bitcoin UTXO-flow analyst reporting
- ERC-20 token-surface and trusted-protocol reporting

### Video

- `video/crypto-profiler-demo.mp4`

The MP4 is a short, silent, 60 to 90 second demo reel assembled from the same screenshots in the recommended walkthrough order.

## How To Regenerate

Run:

```bash
python3 scripts/generate_demo_media.py
```

The generator uses:

- the checked-in sample report files under `docs/sample-reports/`
- local headless Chrome to render styled HTML scenes into PNGs
- `ffmpeg` to assemble the MP4

## Recommended Use

- use the screenshots in portfolio pages, GitHub project writeups, or recruiter packets
- use the MP4 as a silent reel during outreach or as a lightweight demo attachment
- pair the media with `docs/DEMO-WALKTHROUGH.md` for the live talk track
