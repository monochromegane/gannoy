%define _binaries_in_noarch_packages_terminate_build 0

Summary: Approximate nearest neighbor search server and dynamic index written in Golang.
Name:    gannoy
Version: 0.0.1
Release: 1
License: MIT
Group:   Applications/System
URL:     https://github.com/monochromegane/gannoy

Source0:   %{name}-%{version}
Source1:   %{name}-converter-%{version}
Source2:   %{name}-server-%{version}
Source3:   %{name}-db-%{version}
Source4:   %{name}-server.toml
Source5:   %{name}-db.toml
Source6:   %{name}-db.service
Source7:   %{name}-db.logrotate
BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root

%description
%{summary}

%prep

%build

%install
%{__rm} -rf %{buildroot}
%{__mkdir} -p %{buildroot}/var/run/gannoy
%{__mkdir} -p %{buildroot}/var/lib/gannoy
%{__mkdir} -p %{buildroot}/var/log/gannoy
%{__install} -Dp -m0755 %{SOURCE0} %{buildroot}/usr/bin/%{name}
%{__install} -Dp -m0755 %{SOURCE1} %{buildroot}/usr/bin/%{name}-converter
%{__install} -Dp -m0755 %{SOURCE2} %{buildroot}/usr/bin/%{name}-server
%{__install} -Dp -m0755 %{SOURCE3} %{buildroot}/usr/bin/%{name}-db
%{__install} -Dp -m0755 %{SOURCE4} %{buildroot}/etc/%{name}/%{name}-server.toml
%{__install} -Dp -m0755 %{SOURCE5} %{buildroot}/etc/%{name}/%{name}-db.toml
%{__install} -Dp -m0755 %{SOURCE6} %{buildroot}/usr/lib/systemd/system/%{name}-db.service
%{__install} -Dp -m0755 %{SOURCE7} %{buildroot}/etc/logrotate.d/%{name}-db

%clean
%{__rm} -rf %{buildroot}

%post

%files
%defattr(-,root,root)
/usr/bin/%{name}
/usr/bin/%{name}-converter
/usr/bin/%{name}-server
/usr/bin/%{name}-db
%config(noreplace) /etc/%{name}/%{name}-server.toml
%config(noreplace) /etc/%{name}/%{name}-db.toml
%config(noreplace) /etc/logrotate.d/%{name}-db
%config(noreplace) /usr/lib/systemd/system/%{name}-db.service
%dir /var/run/gannoy
%dir /var/lib/gannoy
%dir /var/log/gannoy
