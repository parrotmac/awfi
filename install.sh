#!//usr/bin/env bash
set -euo pipefail

# This script install the latest version of the package from GitHub Releases

verbose=false

verbose_print() {
  if [ "$verbose" = true ]; then
    echo "$@"
  fi
}

print_green() {
  echo -e "\033[0;32m$@\033[0m"
}

print_error() {
  echo "$@" >&2
}

prerequisites() {
  if ! command -v curl &> /dev/null; then
    print_error "curl could not be found. Please install it first."
    exit 1
  fi

  if ! command -v jq &> /dev/null; then
    print_error "jq could not be found. Please install it first."
    exit 1
  fi
}

install() {
  local latest_release_url="https://api.github.com/repos/parrotmac/awfi/releases/latest"
  verbose_print "Downloading the latest release from $latest_release_url"

  local download_url
  local file_name
  local target_dir

  local os_name
  case "$(uname)" in
    Linux*) os_name="Linux" ;;
    Darwin*) os_name="Darwin" ;;
    *) print_error "Unsupported OS: $(uname)"; exit 1 ;;
  esac

  local arch_name
  case "$(uname -m)" in
    x86_64) arch_name="x86_64" ;;
    i386) arch_name="i386" ;;
    aarch64) arch_name="arm64" ;;
    arm64) arch_name="arm64" ;;
    *) print_error "Unsupported architecture: $(uname -m)"; exit 1 ;;
  esac

  verbose_print "Detected OS & Architecture: $os_name-$arch_name ($(uname -m))"

  # Get the latest release download URL
  releases=$(curl -fsSL "$latest_release_url" \
        -H "Accept: application/vnd.github+json" \
        -H "X-GitHub-Api-Version: 2022-11-28" )
  file_name="awfi_${os_name}_${arch_name}.tar.gz"
  download_url=$(echo "$releases" | jq -r '.assets[] | select(.name | test("'"$file_name"'$")) | .url')
  if [ -z "$download_url" ]; then
    print_error "Failed to find the download URL for the latest release."
    exit 1
  fi
  target_dir="$HOME/bin"
  if [ ! -d "$target_dir" ]; then
    verbose_print "Creating target directory: $target_dir"
    mkdir -p "$target_dir"
  fi

  verbose_print "Downloading $file_name from $download_url..."

  # Download and extract the package
  temp_dir=$(mktemp -d)
  trap 'rm -rf "$temp_dir"' EXIT
  curl -fsSLo \
   "${temp_dir}/$file_name" \
    -H "Accept:application/octet-stream" \
    "$download_url"

  tar -xzf "${temp_dir}/$file_name" -C "$temp_dir"

  # Make the binary executable
  chmod +x "${temp_dir}/awfi"

  # Move the binary to the target directory
  mkdir -p "$target_dir"
  mv "${temp_dir}/awfi" "$target_dir/"

  print_green "Installation complete! awfi is now installed in $target_dir."
  if command -v awfi &> /dev/null; then
    print_green "You are now ready to use awfi! You can run it by typing 'awfi' in your terminal."
  else
    print_green "  🪄  Ensure \$HOME is in your \$PATH to run awfi."
    print_green "     Add the following line to your shell configuration file (e.g., ~/.bashrc or ~/.zshrc):"
    print_green "     export PATH=\"\$PATH:\$HOME/bin\""
    print_green "     Then, either restart your terminal or run:"
    print_green "     source ~/.bashrc"
    print_green "     or"
    print_green "     source ~/.zshrc"
    print_green "     to apply the changes."
  fi
}

main() {
  while getopts ":v" opt; do
    case $opt in
      v) verbose=true ;;
      \?) print_error "Invalid option: -$OPTARG" ;;
      :) print_error "Option -$OPTARG requires an argument." ;;
    esac
  done
  shift $((OPTIND - 1))
  if [ "$#" -ne 0 ]; then
    print_error "No arguments are expected."
    exit 1
  fi

  prerequisites
  install
}

main "$@"