import os
from pyarrow.parquet import read_metadata
from tqdm import tqdm
from pathlib import Path

# Your specific SSD directory
DATA_DIR = Path("/your_local_drive/crypto-profiler/eth-data-tmp")


def verify_and_purge(directory):
    # Get all .parquet files, ignoring macOS metadata '._' files
    all_files = list(directory.glob("*.parquet"))
    real_files = [f for f in all_files if not f.name.startswith("._")]

    print(f"🔍 Scanning {len(real_files)} shards in {directory.name}...")

    deleted_count = 0

    for file_path in tqdm(real_files, desc="Checking Integrity"):
        try:
            # Attempt to read the footer
            read_metadata(file_path)
        except Exception:
            # If magic bytes are missing or file is truncated, delete it
            os.remove(file_path)
            deleted_count += 1

    print("\n" + "—" * 40)
    print(f"✅ Healthy Shards: {len(real_files) - deleted_count}")
    print(f"🗑️  Deleted Corrupted: {deleted_count}")
    print(f"🚀 Space cleared for the Go resume script.")
    print("—" * 40)


if __name__ == "__main__":
    if DATA_DIR.exists():
        verify_and_purge(DATA_DIR)
    else:
        print(f"❌ Error: Path not found: {DATA_DIR}")