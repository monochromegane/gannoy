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
BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root

%description
%{summary}

%prep

%build

%install
%{__rm} -rf %{buildroot}
%{__install} -Dp -m0755 %{SOURCE0} %{buildroot}/usr/local/bin/%{name}
%{__install} -Dp -m0755 %{SOURCE1} %{buildroot}/usr/local/bin/%{name}-converter
%{__install} -Dp -m0755 %{SOURCE2} %{buildroot}/usr/local/bin/%{name}-server
%{__install} -Dp -m0755 %{SOURCE3} %{buildroot}/usr/local/bin/%{name}-db
%{__install} -Dp -m0755 %{SOURCE4} %{buildroot}/etc/%{name}/%{name}-server.toml
%{__install} -Dp -m0755 %{SOURCE5} %{buildroot}/etc/%{name}/%{name}-db.toml

%clean
%{__rm} -rf %{buildroot}

%post

%files
%defattr(-,root,root)
/usr/local/bin/%{name}
/usr/local/bin/%{name}-converter
/usr/local/bin/%{name}-server
/usr/local/bin/%{name}-db
/etc/%{name}/%{name}-server.toml
/etc/%{name}/%{name}-db.toml
