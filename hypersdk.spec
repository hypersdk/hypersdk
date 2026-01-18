Name:           hypersdk
Version:        0.2.0
Release:        1%{?dist}
Summary:        Multi-cloud VM export and management toolkit

License:        LGPL-3.0-or-later
URL:            https://github.com/ssahani/hypersdk
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21
BuildRequires:  systemd-rpm-macros
BuildRequires:  git

Requires:       systemd

%description
hypersdk is a high-performance, daemon-based VM export and management system
with 51+ REST API endpoints and optional web dashboard. It offers:
- Interactive CLI (hyperexport) for manual exports with beautiful terminal UI
- Background daemon (hypervisord) with comprehensive REST API (51+ endpoints)
- Control CLI (hyperctl) for daemon management and job submission
- Web dashboard for browser-based VM management and console access
- Complete libvirt/KVM integration (domains, snapshots, networks, storage)
- VMware vSphere export capabilities with parallel downloads
- Job scheduling, webhooks, and monitoring (Prometheus metrics)
- API-only mode (--disable-web) for security-conscious deployments
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
install -Dm644 systemd/hypervisord.service %{buildroot}%{_unitdir}/hypervisord.service

# Install configuration
install -Dm644 examples/config.yaml.example %{buildroot}%{_sysconfdir}/hypersdk/config.yaml

# Install web dashboard (to working directory for daemon access)
install -dm755 %{buildroot}%{_sharedstatedir}/hypersdk/web/dashboard
install -Dm644 web/dashboard/index.html %{buildroot}%{_sharedstatedir}/hypersdk/web/dashboard/index.html
install -Dm644 web/dashboard/vm-console.html %{buildroot}%{_sharedstatedir}/hypersdk/web/dashboard/vm-console.html

# Create data and log directories
install -dm755 %{buildroot}%{_sharedstatedir}/hypersdk
install -dm755 %{buildroot}%{_localstatedir}/log/hypersdk

# Install documentation
install -Dm644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -Dm644 CHANGELOG.md %{buildroot}%{_docdir}/%{name}/CHANGELOG.md
install -Dm644 SECURITY.md %{buildroot}%{_docdir}/%{name}/SECURITY.md
install -Dm644 docs/GETTING-STARTED.md %{buildroot}%{_docdir}/%{name}/GETTING-STARTED.md
install -Dm644 docs/API_ENDPOINTS.md %{buildroot}%{_docdir}/%{name}/API_ENDPOINTS.md
install -Dm644 docs/PROJECT-SUMMARY.md %{buildroot}%{_docdir}/%{name}/PROJECT-SUMMARY.md
install -Dm644 examples/example-vm-export.yaml %{buildroot}%{_docdir}/%{name}/examples/example-vm-export.yaml
install -Dm644 examples/example-batch-export.yaml %{buildroot}%{_docdir}/%{name}/examples/example-batch-export.yaml

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
%doc README.md
%doc %{_docdir}/%{name}/CHANGELOG.md
%doc %{_docdir}/%{name}/SECURITY.md
%doc %{_docdir}/%{name}/GETTING-STARTED.md
%doc %{_docdir}/%{name}/API_ENDPOINTS.md
%doc %{_docdir}/%{name}/PROJECT-SUMMARY.md
%doc %{_docdir}/%{name}/examples/example-vm-export.yaml
%doc %{_docdir}/%{name}/examples/example-batch-export.yaml
%{_bindir}/hyperexport
%{_bindir}/hypervisord
%{_bindir}/hyperctl
%{_unitdir}/hypervisord.service
%dir %{_sysconfdir}/hypersdk
%config(noreplace) %{_sysconfdir}/hypersdk/config.yaml
%attr(0755,hypersdk,hypersdk) %dir %{_sharedstatedir}/hypersdk
%attr(0755,hypersdk,hypersdk) %dir %{_sharedstatedir}/hypersdk/web
%attr(0755,hypersdk,hypersdk) %dir %{_sharedstatedir}/hypersdk/web/dashboard
%attr(0644,hypersdk,hypersdk) %{_sharedstatedir}/hypersdk/web/dashboard/index.html
%attr(0644,hypersdk,hypersdk) %{_sharedstatedir}/hypersdk/web/dashboard/vm-console.html
%attr(0755,hypersdk,hypersdk) %dir %{_localstatedir}/log/hypersdk

%changelog
* Mon Jan 20 2026 Susant Sahani <ssahani@gmail.com> - 0.2.0-1
- Phase 2 release - Production ready
- Added 51+ REST API endpoints for complete VM management
- Added web dashboard (index.html, vm-console.html)
- Added libvirt/KVM integration (domains, snapshots, networks, volumes)
- Added console access features (VNC, serial, screenshots)
- Added job scheduling and webhook support
- Added Prometheus metrics integration
- Added --disable-web flag for API-only deployments
- Security enhancements: TLS validation, path traversal protection, timing-safe auth
- Updated systemd service to run as hypersdk user (not root)
- Added comprehensive documentation (CHANGELOG.md, SECURITY.md, API_ENDPOINTS.md)
- Fixed systemd service paths and permissions
- Improved error handling and logging throughout

* Sat Jan 17 2026 Susant Sahani <ssahani@gmail.com> - 0.1.0-1
- Initial package release
- Multi-cloud provider architecture (vSphere production-ready)
- Beautiful terminal UI with pterm
- REST JSON API daemon (6 core endpoints)
- Configuration file support (YAML)
- Systemd integration
- Parallel downloads with worker pools
- Resumable downloads with retry logic
- Interactive VM selection
- Batch job processing
- Comprehensive logging
