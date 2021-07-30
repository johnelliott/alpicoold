# Alpicoold

Apple HomeKit bridge for Alpicool portable cooler style fridges

## What is this?
This is a personal hack project for my little fridge.

Alpicoold:
- Is a system daemon
- Runs via systemd
- Intended for Raspberry Pi or Linux
- Uses Bluetooth and IP networking.
- Interacts with a fridge on the Bluetooth side
- Interacts with Apple HomeKit on the IP side
- Acts as a HomeKit "bridge"
- Exposes thermostat, toggle switches, and a webcam to HomeKit
- Uses Ansible as the main way to deploy code to the target host
- Looks as experimental and rough as it is :)

See the k25 package for the Bluetooth characteristic data frames used to interact with the fridge. The protocol is inferred from packet sniffing the manufacturer's app traffic using Wireshark.


## Installs
Deployment is via an Ansible playbook called deploy, with another for installing the camera and ffmpeg loopback device.

This playbook uses make and cross-compiles from x86/Darwin to arm/linux via the musl cross-compiler before copying the executable to the Pi.

1. Install the cross-compiler on the mac
1. Set up Ansible inventory and variables
1. Install the camera requirements on the pi manually or via the camera role (camera uses [hkcam](https://github.com/brutella/hkcam))
1. Run the deploy playbook to install and run the daemon

## Deploys
```bash
ansible-playbook -i ~/inventory.yml ansible/deploy.yml -l pizero2 -e'loglevel=info'
```

## Monitoring Bluetooth on Linux
Some commands to remember for monitoring Bluetooth on Raspberry Pi:
```bash
sudo btmon
sudo bluetoothctl
sudo dbus-monitor --system "type=error"
```
