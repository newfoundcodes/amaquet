---
title: "Installation"
description: "Download and install Amaquet on Ubuntu, Debian, Windows, or macOS, or build it from source with a secure first-start configuration."
---

This guide covers the supported release packages and source installation. Amaquet is a standalone Go database server; its packages do not install Redis or another database. The Debian package includes an optional systemd unit, but it does not start or enable it automatically.

## Choose an installation method

Use a release package for the quickest installation:

| Platform | Package | Architectures | Notes |
| --- | --- | --- | --- |
| Ubuntu/Debian | `.deb` | `amd` (32-bit Intel), `amd64` (64-bit x86), `arm64` | Install with `apt` or `dpkg`. |
| Windows | `.msi` | `amd64` (64-bit x86) | Windows Installer provides repair and uninstall. |
| macOS | `.pkg` | Intel, Apple Silicon, or universal | The universal package runs on either Mac architecture. |

Build from source when you need a development build, a platform not covered by the release packages, or control over build flags and installation paths. A source build writes binaries to the repository and does not copy anything into a system directory.

## Download a release

Download packages from the [Amaquet GitHub Releases page](https://github.com/newfoundcodes/amaquet/releases). Open the desired release and download the appropriate file from its **Assets** section.

The packaging workflow uses these filenames, where `VERSION` is the release version without the leading `v`:

| File | Contents |
| --- | --- |
| `amaquet_VERSION_amd.deb` | Debian package for 32-bit Intel (`i386`). |
| `amaquet_VERSION_amd64.deb` | Debian package for 64-bit Intel/AMD (`amd64`). |
| `amaquet_VERSION_arm64.deb` | Debian package for 64-bit ARM (`arm64`). |
| `amaquet-VERSION-windows-amd64.msi` | 64-bit Windows MSI. |
| `amaquet-VERSION-macos-intel.pkg` | Intel macOS package (`x86_64`). |
| `amaquet-VERSION-macos-apple-silicon.pkg` | Apple Silicon macOS package (`arm64`). |
| `amaquet-VERSION-macos-universal.pkg` | Universal macOS package containing both binary slices. |

For version `0.1.0`, examples include `amaquet_0.1.0_amd64.deb`, `amaquet-0.1.0-windows-amd64.msi`, and `amaquet-0.1.0-macos-universal.pkg`.

The GitHub Actions workflow also uploads package files as workflow artifacts. Workflow artifacts are attached to a workflow run and are separate from release assets. If a release has no files under **Assets**, check the corresponding [Actions workflow runs](https://github.com/newfoundcodes/amaquet/actions) or use a release where the artifacts have been attached.

The current packaging workflow does not publish checksum files. Use the official release page and a trusted transport when downloading packages, and verify that the filename and version are the ones you intended.

## Ubuntu and Debian

The Debian packages install four executables under `/usr/bin`, a systemd unit named `amaquetd`, a service configuration under `/etc/amaquet`, and a persistent service data directory at `/var/lib/amaquet`. Installation does not start or enable the service automatically.

### Select the package

Check the operating-system architecture:

```bash
dpkg --print-architecture
```

Choose the matching asset:

| `dpkg --print-architecture` | Asset suffix | Meaning |
| --- | --- | --- |
| `i386` | `_amd.deb` | 32-bit Intel. |
| `amd64` | `_amd64.deb` | 64-bit Intel or AMD. |
| `arm64` | `_arm64.deb` | 64-bit ARM. |

The project calls the 32-bit build `amd` in the filename, but the package metadata uses Debian architecture `i386`. An `amd64` operating system should normally use the `amd64` package; do not choose `amd` merely because the CPU vendor is AMD.

You can also inspect the CPU with:

```bash
uname -m
```

### Install with apt

Download the package, change to its directory, and include `./` so `apt` treats it as a local file:

```bash
cd ~/Downloads
sudo apt install ./amaquet_VERSION_amd64.deb
```

Replace the filename with the asset you downloaded. `apt` installs the package and resolves declared dependencies. Amaquet currently declares no external runtime dependencies, but `apt` is the preferred installer because it reports dependency problems clearly.

To upgrade an existing installation, install the newer package:

```bash
sudo apt install ./amaquet_NEW_VERSION_amd64.deb
```

### Install with dpkg

Use `dpkg` when you want to install the downloaded file directly without asking `apt` to resolve the transaction.

```bash
sudo dpkg -i ./amaquet_VERSION_amd64.deb
```

If `dpkg` reports missing dependencies, let `apt` complete the transaction:

```bash
sudo apt-get -f install
```

### Verify and start

Inspect the package and executables:

```bash
dpkg-query -W -f='${Status} ${Version} ${Architecture}\n' amaquet
command -v amaquet
amaquet -version
amaquet-cli -h
```

The package owns these files and directories:

```text
/usr/bin/amaquet
/usr/bin/amaquet-cli
/usr/bin/amaquet-keygen
/usr/bin/amaquet-restore
/usr/share/amaquet/amaquet.example.json
/usr/share/doc/amaquet/README.md
/usr/share/doc/amaquet/LICENSE
/etc/amaquet/amaquet.json
/lib/systemd/system/amaquetd.service
/var/lib/amaquet/
```

The packaged service runs as the unprivileged `amaquet` system user, binds both listeners to loopback, enables AOF at `/var/lib/amaquet/amaquet.aof`, and creates its bootstrap token at `/var/lib/amaquet/bootstrap-token` on first start. Read the token with `sudo cat /var/lib/amaquet/bootstrap-token`, then create a permanent API key through the authenticated admin API.

### Manage the Debian service

The package installs the unit but does not start it automatically. Use these commands:

```bash
sudo systemctl status amaquetd
sudo systemctl start amaquetd
sudo systemctl stop amaquetd
sudo systemctl enable amaquetd
sudo systemctl disable amaquetd
```

To enable the service and start it immediately:

```bash
sudo systemctl enable --now amaquetd
```

Check startup and runtime logs with:

```bash
sudo journalctl -u amaquetd --no-pager
sudo journalctl -u amaquetd -f
```

After changing `/etc/amaquet/amaquet.json`, restart the service because Amaquet loads configuration at process startup:

```bash
sudo systemctl restart amaquetd
```

The packaged configuration is intentionally loopback-only and does not configure TLS. Keep the listeners local unless you replace the configuration with a reviewed TLS-enabled deployment profile. For a local, foreground installation instead, use:

```bash
mkdir -p "$HOME/amaquet"
amaquet --init-config --config "$HOME/amaquet/amaquet.json"
amaquet --config "$HOME/amaquet/amaquet.json"
```

Initialization prints a bootstrap token once and uses a local `data` directory by default. Save the token immediately. For a managed service, create your own systemd or supervisor configuration after reviewing [Configuration](configuration.md) and [Production deployment](../operations/deployment.md).

### Uninstall

Stop and disable the service before removing the package:

```bash
sudo systemctl disable --now amaquetd
sudo apt remove amaquet
```

Or:

```bash
sudo dpkg -r amaquet
```

Removal deletes package-owned executables and the unit. Debian normally retains `/etc/amaquet/amaquet.json` as a conffile, and the package deliberately preserves the `amaquet` system user and `/var/lib/amaquet` data so that an uninstall does not destroy AOF or administration state. It also does not delete other files created outside the package, such as `./data` or custom TLS material. Back up and remove preserved files separately only when you are certain they are no longer needed.

## Windows MSI

The Windows release currently contains a 64-bit (`amd64`) MSI. It installs into:

```text
C:\Program Files\Amaquet\bin\amaquet.exe
C:\Program Files\Amaquet\bin\amaquet-cli.exe
C:\Program Files\Amaquet\bin\amaquet-keygen.exe
C:\Program Files\Amaquet\bin\amaquet-restore.exe
C:\Program Files\Amaquet\amaquet.example.json
C:\Program Files\Amaquet\README.md
C:\Program Files\Amaquet\LICENSE
```

The MSI does not create a Windows service, scheduled task, or Start Menu shortcut. It adds `C:\Program Files\Amaquet\bin` to the system `PATH`. Open a new PowerShell or Command Prompt window after installation so it receives the updated environment.

### Install through the Windows Installer UI

Use the graphical Windows Installer when you want the normal installation and maintenance prompts.

1. Download `amaquet-VERSION-windows-amd64.msi` from the release **Assets** section.
2. Open the `.msi` file.
3. Approve the administrator prompt.
4. Follow the installer steps.
5. On the final page, confirm that installation completed successfully.
6. Open a new PowerShell window before testing the PATH change.

The MSI embeds the Amaquet logo as its Add/Remove Programs icon, installs per machine under `C:\Program Files\Amaquet`, and uses the standard Windows Installer completion page to report a successful installation.

### Install silently

Run PowerShell as an administrator:

```powershell
msiexec.exe /i .\amaquet-VERSION-windows-amd64.msi /qn /norestart /l*v .\amaquet-install.log
```

The log records Windows Installer details for troubleshooting.

### Repair and verify

Open the MSI again and choose **Repair** when Windows displays maintenance options. The MSI uses standard Windows Installer maintenance behavior and prevents downgrading over a newer version.

Verify the binaries:

```powershell
amaquet.exe -version
amaquet-cli.exe -h
```

To start Amaquet, create a configuration using Windows paths:

```powershell
& 'C:\Program Files\Amaquet\bin\amaquet.exe' --init-config --config "$env:USERPROFILE\amaquet\amaquet.json"
& 'C:\Program Files\Amaquet\bin\amaquet.exe' --config "$env:USERPROFILE\amaquet\amaquet.json"
```

Save the bootstrap token printed during initialization. Press `Ctrl-C` to stop the foreground server.

### Uninstall

Use either supported Windows maintenance interface:

1. Open **Settings → Apps → Installed apps**.
2. Find **Amaquet**.
3. Select the menu button and choose **Uninstall**.
4. Confirm the administrator prompt.

Alternatively, open **Control Panel → Programs and Features**, select **Amaquet**, and choose **Uninstall** or **Repair**.

With the original MSI available, you can uninstall silently:

```powershell
msiexec.exe /x .\amaquet-VERSION-windows-amd64.msi /qn /norestart /l*v .\amaquet-uninstall.log
```

Uninstallation removes files under `C:\Program Files\Amaquet` and removes the Amaquet `bin` entry from the system `PATH`. It does not remove configuration, AOF data, or other files created under your user profile or another directory. Stop the server and back up data first.

## macOS packages

The release provides three packages:

| Package | Use on |
| --- | --- |
| `amaquet-VERSION-macos-intel.pkg` | Intel Macs (`x86_64`). |
| `amaquet-VERSION-macos-apple-silicon.pkg` | Apple Silicon Macs (`arm64`). |
| `amaquet-VERSION-macos-universal.pkg` | Either architecture; contains both slices. |

Check your architecture:

```bash
uname -m
```

The result is `x86_64` for Intel or `arm64` for Apple Silicon. The universal package is the simplest choice when distributing one file to both types of Mac, while an architecture-specific package is smaller.

The package owns:

```text
/usr/local/bin/amaquet
/usr/local/bin/amaquet-cli
/usr/local/bin/amaquet-keygen
/usr/local/bin/amaquet-restore
/usr/local/share/amaquet/amaquet.example.json
/usr/local/share/doc/amaquet/README.md
/usr/local/share/doc/amaquet/LICENSE
```

It does not install a launch daemon, launch agent, shell profile entry, persistent configuration, or running server. The installer displays the Amaquet logo, but the current packages are not signed or notarized with an Apple Developer ID.

### Install through Installer

Use the macOS Installer application when you want to approve the package and installation location interactively.

1. Download the appropriate `.pkg` from the release **Assets** section.
2. Open the package.
3. Select **Install** and approve the administrator prompt.
4. Verify the installation in Terminal.

Because the package is not notarized, macOS may say that Apple cannot verify it. If the file came from the official release page and you trust it:

- In Finder, Control-click the package, select **Open**, and confirm **Open**; or
- Attempt to open it once, then use **System Settings → Privacy & Security → Open Anyway**.

Do not disable Gatekeeper globally.

### Install from Terminal

Use the `installer` command when you want to install the package without opening the graphical Installer application.

```bash
cd ~/Downloads
sudo installer -pkg ./amaquet-VERSION-macos-universal.pkg -target /
```

Use the Intel or Apple Silicon filename when appropriate. The `-target /` argument installs the payload under `/usr/local`.

Verify the receipt and binaries:

```bash
pkgutil --pkg-info com.newfoundcodes.amaquet
command -v amaquet
amaquet -version
amaquet-cli -h
file /usr/local/bin/amaquet
```

For the universal package, `file` should report both `x86_64` and `arm64` Mach-O slices.

### Start Amaquet

The package installs the binaries but does not start a server, so launch Amaquet explicitly with a configuration file.

```bash
mkdir -p "$HOME/amaquet"
amaquet --init-config --config "$HOME/amaquet/amaquet.json"
amaquet --config "$HOME/amaquet/amaquet.json"
```

The default listeners bind to loopback. The initialization command prints a bootstrap token once and uses a relative `data` directory unless the configuration changes it. Review [Configuration](configuration.md) before exposing a listener.

### Uninstall

The package has no separate uninstaller. Stop Amaquet first, then remove only package files:

```bash
sudo rm -f \
  /usr/local/bin/amaquet \
  /usr/local/bin/amaquet-cli \
  /usr/local/bin/amaquet-keygen \
  /usr/local/bin/amaquet-restore

sudo rm -rf \
  /usr/local/share/amaquet \
  /usr/local/share/doc/amaquet
```

Remove the package receipt:

```bash
sudo pkgutil --forget com.newfoundcodes.amaquet
```

`pkgutil --forget` removes only the receipt; it does not delete files. Inspect `/usr/local/share/amaquet` first if you placed custom configuration or data there, because removing that directory removes those files too. The package does not install AOF data, a launch daemon, or a launch agent. Remove separately created files such as `~/amaquet/amaquet.json`, `~/amaquet/data`, or an AOF path only after backing them up.

## What gets installed

All installation methods provide the same four executables:

| Executable | Purpose |
| --- | --- |
| `amaquet` | Database server, native protocol listener, and HTTP administration listener. |
| `amaquet-cli` | Command-line client for native protocol commands and typed-value operations. |
| `amaquet-keygen` | Generates an optional Ed25519 server identity key pair. |
| `amaquet-restore` | Validates and atomically installs a verified AOF checkpoint while the server is stopped. |

The server keeps active values in memory. AOF persistence and control-plane files under `data_dir` are optional, but durable deployments should put them on persistent, protected storage. The Debian service package configures AOF under `/var/lib/amaquet`; source, Windows, and macOS installations do not configure persistence automatically.

## Build from source

Source builds require Go 1.23 or newer. Git is needed to clone the repository. Node.js 22.12 or newer and npm are only required for the documentation site.

```bash
go version
git --version
node --version   # only required for docs
npm --version    # only required for docs
```

Clone and build:

```bash
git clone https://github.com/newfoundcodes/amaquet.git
cd amaquet
make build
```

The equivalent commands are:

```bash
mkdir -p bin
go build -trimpath -o bin/amaquet ./cmd/amaquet
go build -trimpath -o bin/amaquet-cli ./cmd/amaquet-cli
go build -trimpath -o bin/amaquet-keygen ./cmd/amaquet-keygen
go build -trimpath -o bin/amaquet-restore ./cmd/amaquet-restore
```

Verify the output:

```bash
./bin/amaquet -version
./bin/amaquet-cli -h
```

`make build` does not install files into a system directory. Invoke the binaries with `./bin/` paths or add `bin` to a controlled PATH.

## First configuration and verification

Generate a secure local configuration:

```bash
./bin/amaquet --init-config --config ./amaquet.json
```

This writes the JSON file, prints a bootstrap token once, and exits. Save the token immediately and never commit the generated file. The defaults bind the native and admin listeners to loopback, require native authentication, disable AOF, and use `./data` for runtime state.

Start the server:

```bash
./bin/amaquet --config ./amaquet.json
```

Default endpoints:

```text
Native protocol: amaquet://127.0.0.1:13378
Admin API:       http://127.0.0.1:13379
Data directory:  ./data
```

Check health and authenticate with the bootstrap token:

```bash
curl --fail --silent --show-error http://127.0.0.1:13379/api/health

export AMAQUET_BOOTSTRAP_TOKEN='paste-the-token-from-initialization'
./bin/amaquet-cli \
  -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_BOOTSTRAP_TOKEN" \
  PING '{}'
```

For a data-plane check:

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 -api-key "$AMAQUET_BOOTSTRAP_TOKEN" \
  SET '{"key":"install:check","type":"utf8_string","value":"Amaquet is running"}'

./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 -api-key "$AMAQUET_BOOTSTRAP_TOKEN" \
  GET '{"key":"install:check"}'
```

Create permanent API keys through the authenticated admin API, using the narrowest role and permissions required. Raw secrets are shown only once; after the first admin API key is created, bootstrap authentication is disabled in persisted administration state. See [Admin API](../reference/admin-api.md).

## Network access and TLS

The generated configuration is loopback-only. Before exposing a listener, read [Configuration](configuration.md) and [Production deployment](../operations/deployment.md):

- Use a specific private bind address instead of `0.0.0.0` when possible.
- Enable `protocol.tls` and use `amaquets://` for native traffic.
- Enable `admin.tls` and protect its private key.
- Keep `protocol.require_auth` and `admin.metrics_require_auth` enabled.
- Leave `protocol.allow_insecure_auth` and `admin.allow_insecure` disabled unless the network is deliberately isolated.
- Restrict firewall access to application and management networks.

Generate the optional Ed25519 protocol identity separately from TLS certificates:

```bash
mkdir -p keys
./bin/amaquet-keygen \
  --public ./keys/amaquet-public.pem \
  --private ./keys/amaquet-private.pem
```

This key pair does not replace an X.509 certificate or encrypt traffic.

## Docker installation

The repository includes Compose files. Start the development deployment with:

```bash
docker compose build
docker compose up -d
docker compose logs -f amaquet
```

The Compose configuration persists `/app/data` in the `amaquet-data` volume. Do not remove that volume if AOF data or administrator state must survive container replacement. See [Docker deployment](../operations/docker.md).

## Documentation tools

Node.js and npm are needed only for the documentation site:

```bash
make docs-install
make docs-build
make docs-serve
```

`make docs-build` validates source-derived inventories, checks headings, builds the static site, and checks generated links. `make docs-serve` starts Astro; it does not start Amaquet.

## Upgrades, backups, and troubleshooting

Before replacing a binary or changing persistence settings, stop Amaquet cleanly and back up the AOF and `data_dir`, including `admin.json`, bootstrap-token material, upload state, and identity files. Never copy over a live AOF; use the administration persistence endpoint and `amaquet-restore` as described in [Persistence and recovery](../concepts/persistence.md) and [Backup and recovery](../operations/backups.md).

| Symptom | Likely action |
| --- | --- |
| `Failed to load configuration` | Validate the JSON and check the `--config` path. |
| `invalid port` | Check that the port is between 1 and 65535 and is not already in use. |
| `TLS requires ...` | Provide readable certificate and key files. |
| `authenticated Amaquet on a non-loopback address requires TLS` | Enable native TLS or deliberately configure `protocol.allow_insecure_auth`; TLS is recommended. |
| CLI connection refused | Confirm the server is running and the URI port matches `protocol.port`. |
| Authentication failure | Use the initialization token, current `data_dir/bootstrap-token`, or a permanent API key. |
| macOS cannot verify the package | The current package is not notarized. Confirm its source, then use Finder **Open** or **Privacy & Security → Open Anyway**. |
| Windows cannot find `amaquet` | Open a new terminal after installation so it receives the updated system PATH, then confirm that `C:\Program Files\Amaquet\bin` is present. |

Continue with [Quick start](quickstart.md), [Configuration](configuration.md), [Authentication and identity](../protocol/authentication.md), and [Production deployment](../operations/deployment.md).
