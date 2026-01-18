Name:           hypersdk
Version:        0.0.1
Release:        1%{?dist}
Summary:        High-performance vSphere VM export toolkit

License:        LGPL-3.0-or-later
URL:            https://github.com/ssahani/hypersdk
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21
BuildRequires:  systemd-rpm-macros
BuildRequires:  git

Requires:       systemd

%description
hypersdk is a high-performance, daemon-based vSphere VM export system. It offers:
- Interactive CLI (hyperexport) for manual exports with beautiful terminal UI
- Background daemon (hypervisord) with REST API for automation
- Control CLI (hyperctl) for daemon management and job submission
- Parallel downloads with configurable worker pools
- YAML/JSON configuration support for batch operations

%prep
%setup -q

%build
# Build all binaries
go build -v -o hyperexport ./cmd/hyperexport
go build -v -o hypervisord ./cmd/hypervisord
go build -v -o hyperctl ./cmd/hyperctl

%install
# Install binaries
install -Dm755 hyperexport %{buildroot}%{_bindir}/hyperexport
install -Dm755 hypervisord %{buildroot}%{_bindir}/hypervisord
install -Dm755 hyperctl %{buildroot}%{_bindir}/hyperctl

# Install systemd service
install -Dm644 hypervisord.service %{buildroot}%{_unitdir}/hypervisord.service

# Install configuration
install -Dm644 config.yaml.example %{buildroot}%{_sysconfdir}/hypersdk/config.yaml

# Create data directory
install -dm755 %{buildroot}%{_sharedstatedir}/hypersdk
install -dm755 %{buildroot}%{_localstatedir}/log/hypersdk

# Install documentation
install -Dm644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -Dm644 GETTING-STARTED.md %{buildroot}%{_docdir}/%{name}/GETTING-STARTED.md
install -Dm644 examples/example-vm-export.yaml %{buildroot}%{_docdir}/%{name}/example-vm-export.yaml
install -Dm644 examples/example-batch-export.yaml %{buildroot}%{_docdir}/%{name}/example-batch-export.yaml

%pre
# Create system user for the daemon if it doesn't exist
getent group hypersdk >/dev/null || groupadd -r hypersdk
getent passwd hypersdk >/dev/null || \
    useradd -r -g hypersdk -d %{_sharedstatedir}/hypersdk \
    -s /sbin/nologin -c "hypersdk daemon user" hypersdk
exit 0

%post
%systemd_post hypervisord.service

# Set ownership
chown -R hypersdk:hypersdk %{_sharedstatedir}/hypersdk
chown -R hypersdk:hypersdk %{_localstatedir}/log/hypersdk

if [ $1 -eq 1 ]; then
    # First install
    echo "hypersdk installed successfully!"
    echo "Edit /etc/hypersdk/config.yaml with your vCenter credentials"
    echo "Start the daemon: systemctl start hypervisord"
    echo "Enable auto-start: systemctl enable hypervisord"
fi

%preun
%systemd_preun hypervisord.service

%postun
%systemd_postun_with_restart hypervisord.service

if [ $1 -eq 0 ]; then
    # Uninstall
    userdel hypersdk 2>/dev/null || true
    groupdel hypersdk 2>/dev/null || true
fi

%files
%license LICENSE
%doc README.md GETTING-STARTED.md
%doc %{_docdir}/%{name}/example-vm-export.yaml
%doc %{_docdir}/%{name}/example-batch-export.yaml
%{_bindir}/hyperexport
%{_bindir}/hypervisord
%{_bindir}/hyperctl
%{_unitdir}/hypervisord.service
%dir %{_sysconfdir}/hypersdk
%config(noreplace) %{_sysconfdir}/hypersdk/config.yaml
%attr(0755,hypersdk,hypersdk) %dir %{_sharedstatedir}/hypersdk
%attr(0755,hypersdk,hypersdk) %dir %{_localstatedir}/log/hypersdk

%changelog
* Sat Jan 17 2026 Susant Sahani <ssahani@gmail.com> - 0.0.1-1
- Initial package release (alpha)
- Multi-cloud provider architecture (vSphere today, AWS/Azure/GCP next)
- Beautiful terminal UI with pterm
- REST JSON API daemon
- Configuration file support (YAML)
- Systemd integration
- Parallel downloads with worker pools
- Resumable downloads with retry logic
- Interactive VM selection
- Batch job processing
- Comprehensive logging
