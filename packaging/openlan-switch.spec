Name: openlan-switch
Version: 5.2.7
Release: 1%{?dist}
Summary: OpenLan's Project Software
Group: Applications/Communications
License: Apache 2.0
URL: https://github.com/danieldin95/openlan-go
BuildRequires: go
Requires: net-tools, iptables, iputils

%define _venv /opt/openlan-utils/env
%define _source_dir ${RPM_SOURCE_DIR}/openlan-go-%{version}

%description
OpenLan's Project Software

%build
cd %_source_dir && make linux/switch

%install
mkdir -p %{buildroot}/usr/bin
cp %_source_dir/build/openlan-switch %{buildroot}/usr/bin

mkdir -p %{buildroot}/etc/openlan/switch
cp %_source_dir/packaging/resource/ctrl.json.example %{buildroot}/etc/openlan/switch
cp %_source_dir/packaging/resource/switch.json.example %{buildroot}/etc/openlan/switch
mkdir -p %{buildroot}/etc/sysconfig/openlan
cp %_source_dir/packaging/resource/switch.cfg %{buildroot}/etc/sysconfig/openlan

mkdir -p %{buildroot}/usr/lib/systemd/system
cp %_source_dir/packaging/resource/openlan-switch.service %{buildroot}/usr/lib/systemd/system

mkdir -p %{buildroot}/var/openlan
cp -R %_source_dir/packaging/resource/ca %{buildroot}/var/openlan
cp -R %_source_dir/packaging/script %{buildroot}/var/openlan
cp -R %_source_dir/switch/public %{buildroot}/var/openlan

%pre
firewall-cmd --permanent --zone=public --add-port=10000/tcp --permanent || {
  echo "YOU NEED ALLOW TCP PORT:10000."
}
firewall-cmd --permanent --zone=public --add-port=10002/udp --permanent || {
  echo "YOU NEED ALLOW UDP PORT:10000."
}
firewall-cmd --permanent --zone=public --add-port=10002/tcp --permanent || {
  echo "YOU NEED ALLOW TCP PORT:10000."
}
firewall-cmd --reload || :

%files
%defattr(-,root,root)
/etc/sysconfig/*
/etc/openlan/*
/usr/bin/*
/usr/lib/systemd/system/*
/var/openlan

%clean
rm -rf %_env
