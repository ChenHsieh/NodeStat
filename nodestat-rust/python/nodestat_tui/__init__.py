"""NodeStat TUI - Modern cluster monitoring interface."""

__version__ = "1.0.0"

import subprocess
import sys
import shutil
from pathlib import Path

def find_binary():
    """Find the nodestat binary."""
    # First try to find the binary in the same directory as this module
    module_dir = Path(__file__).parent
    binary_path = module_dir / "nodestat"
    
    if binary_path.exists() and binary_path.is_file():
        return str(binary_path)
    
    # Fallback to system PATH
    binary_path = shutil.which("nodestat")
    if binary_path:
        return binary_path
    
    raise FileNotFoundError(
        "nodestat binary not found. Make sure it's installed correctly."
    )

def main():
    """Main entry point that calls the Rust binary."""
    try:
        binary_path = find_binary()
        # Forward all arguments to the Rust binary
        result = subprocess.run([binary_path] + sys.argv[1:], check=False)
        sys.exit(result.returncode)
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)

if __name__ == "__main__":
    main()