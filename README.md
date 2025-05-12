# Software-Based Solution for Air-Gap Security

## What?

The software-based security solution is initially developed for a school project. That repo is currently private so this readme doesn't include a link to it.

This project aims to provide security for air-gapped systems by monitoring and preventing activities that are suspicious in an air-gapped context. Examples:

- Check for programs that perform queries to check for internet access. (common behavior for malware that uses exfiltration through USB drives - e.g. USBCulprit)
- Monitor for assembly instructions that perform non-temporal access. (another common behavior for malware that uses electromagnetic covert channels - e.g. RAMBO, Bitjabber)

## How to Use

### Using Releases
- Download the executable from releases. Now you can transfer the executable to your air-gapped system in any way you want.
- Note: The project is developing a security system to block potential malware infections caused by untrusted USB drivers, while ensuring the system itself can be reliably deployed via USB.

### Build from Source
- Download the source code and navigate to `cmd/`
- Download `make` (we're looking for a way to make the compilation not dependent on `make` which is a GNU project
- You should see a file called `makefile`, if you can't see it, make sure that you are in the right directory and that the file is not moved somewhere else.
- Before jumping to the next step, see [Create Config File](#create-config-file)
- Run `make build` to build for your OS, run `make build-all` to compile for Linux, Darwin, and Windows.
- After building the program, you should see the executable files under `build/` in `cmd/`. If you move the executable anywhere other than this directory, the program won't work because it won't be able to find the config file under the path ../../configs/ relative to the program's location. If you want to access the program from other locations then you should create a symbolic link.
#### Create Config File
The program looks for the main config file under `configs/config.yaml`. However, this file is not included in the source on Git. You should use the `configs/config_template.yaml` file to create your own `configs/config.yaml` file instead. Good news is, all you need to do is rename `config_template.yaml` to `config.yaml`. If you want to change the default config, then you can change the config accordingly. You can refer to [Configuration Settings](configuration-settings)

## For Developers and Technical People

### What is a software-based solution

The sofware-based solution can be anything that does not rely on hardware capabilities as this project is created to overcome those restrictions. Some of these restrictions are:

- SDRs (used for monitoring frequencies) that scan frequency ranges can be bypassed as they are not fast enough to pick up on suspicious activity on each frequency. (look: NoiseHopper)
- They cost sweet-sweet money. (faraday cages, unidirectional data diodes)

Currently, we only have an antivirus product under cmd/antivirus/. This product aims to rely on behavior based detection such as suspicious memory consumption and suspicious cache misses
rather than static analysis such as malware signatures.

Theoretically, a softare-based prevention system that ensures the CPU consumption levels remain the same so that a malware cannot control electromagnetic emanations.
Again, theoretically the techniques that are used for preventing side-channel attacks in cryptography can be considered for this project in general.

[Protecting Against Side Channel Attacks by RocketMeUpCybersecurity](https://medium.com/@RocketMeUpCybersecurity/hardware-security-protecting-against-side-channel-and-fault-injection-attacks-a4dc9de8cedc)

[What are Side Channel Attacks and How You Defend Against Them by Wnesecurity](https://wnesecurity.com/what-are-side-channel-attacks-and-how-can-you-defend-against-them/)

### Development Guidelines

#### Programming Language
- This is a Go project (v1.24).
- Go was chosen because of its safety, speed, and subjectively, developer friendly syntax and features
- The initial developers had no prior Go knowledge so do expect funky Go code.
- Download and install go from here https://go.dev/doc/install
- The code must be formatted according to `gofmt`.

#### Project Structure
- Each new application (i.e. executable) has its own directory under cmd/<application_name>
- Any library that is used in the project are under internal/<library_name>
- Any unit test is under internal/<library_name>/<test_name>.go (e.g. internal/config/loader_test.go)
- Any integration test is under cmd/<application_name>/<test_name>.go (e.g. cmd/antivirus/dashboard_test.go)

#### Testing
- Each module (file) in internal/<library_name> (business logic) should have unit tests that are meaningful and helpful for finding potential bugs.
- Each application in cmd/<application_name> (entry points) should have integration tests that are meaningful and helpful for finding bugs.

#### Configuration Settings
This is the current default config
```yaml
version: 0.1.0
cpu_logger: json
```

**version:** current program version
**cpu_logger:** logger type for the cpu logger to use there are currently 2 types:
- json: json format provided by Go's log/slog package
- pretty: human readable format