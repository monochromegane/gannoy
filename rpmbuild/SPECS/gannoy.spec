%define _binaries_in_noarch_packages_terminate_build 0
%define gannoy_user   gannoy
%define gannoy_group  %{gannoy_user}
%define gannoy_confdir %{_sysconfdir}/gannoy
%define gannoy_home    %{_localstatedir}/lib/gannoy
%define gannoy_logdir  %{_localstatedir}/log/gannoy
%define gannoy_rundir  %{_localstatedir}/run/gannoy

Summary: Approximate nearest neighbor search server and dynamic index written in Golang.
Name:    gannoy
Version: 0.0.2
Release: 2
License: MIT
Group:   Applications/System
URL:     https://github.com/monochromegane/gannoy

Source0:   %{name}-%{version}
Source1:   %{name}-converter-%{version}
Source2:   %{name}-db-%{version}
Source3:   %{name}-db.toml
Source4:   %{name}-db.service
Source5:   %{name}-db.logrotate
BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root

%{?systemd_requires}
BuildRequires: systemd

%description
%{summary}

%prep

%build

%install
%{__rm} -rf %{buildroot}
%{__mkdir} -p %{buildroot}%{gannoy_rundir}
%{__mkdir} -p %{buildroot}%{gannoy_home}
%{__mkdir} -p %{buildroot}%{gannoy_logdir}
%{__install} -Dp -m0755 %{SOURCE0} %{buildroot}/usr/bin/%{name}
%{__install} -Dp -m0755 %{SOURCE1} %{buildroot}/usr/bin/%{name}-converter
%{__install} -Dp -m0755 %{SOURCE2} %{buildroot}/usr/bin/%{name}-db
%{__install} -Dp -m0644 %{SOURCE3} %{buildroot}%{gannoy_confdir}/%{name}-db.toml
%{__install} -Dp -m0644 %{SOURCE4} %{buildroot}/usr/lib/systemd/system/%{name}-db.service
%{__install} -Dp -m0644 %{SOURCE5} %{buildroot}/etc/logrotate.d/%{name}-db

%clean
%{__rm} -rf %{buildroot}

%pre
%{_sbindir}/useradd -c "Gannoy user" -s /bin/false -r -d %{gannoy_home} %{gannoy_user} 2>/dev/null || :

%post
%systemd_post %{name}-db.service
systemctl enable %{name}-db.service

%preun
%systemd_preun %{name}-db.service

%postun
%systemd_postun %{name}-db.service

%files
%defattr(-,root,root)
/usr/bin/%{name}
/usr/bin/%{name}-converter
/usr/bin/%{name}-db
%config(noreplace) %{gannoy_confdir}/%{name}-db.toml
%config(noreplace) /etc/logrotate.d/%{name}-db
%config(noreplace) /usr/lib/systemd/system/%{name}-db.service
%attr(-,%{gannoy_user},%{gannoy_group}) %dir %{gannoy_rundir}
%attr(-,%{gannoy_user},%{gannoy_group}) %dir %{gannoy_home}
%attr(-,%{gannoy_user},%{gannoy_group}) %dir %{gannoy_logdir}
