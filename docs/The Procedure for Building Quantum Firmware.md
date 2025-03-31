# The Procedure for Creating Quantum Switching Build

## INSTRUCTIONS

- **DON’T MISS ANY STEPS.** IF YOU MISS, THE FIRMWARE WILL BEHAVE UNEXPECTEDLY.
- This document is only for **“Alleycat3”**, **“Aldrin2”**, and **“PonCat3”** platforms only.
- Using this document, a developer can easily create Quantum switch firmware.
- **Linux** as your host machine is required, and Linux expertise is necessary.

### Required Tools for Firmware Building

The following software packages are required:

- GNU coreutils
- lib32 support on `x86_64` because the `armv7-marvell-linux-gnueabi` toolchain is `x86_32` **only**
- lib32-fakeroot
- fakeroot
- tar
- gunzip
- sed
- cpio
- jacksum

### Custom Changes

Below are changes required for the firmware building setup:

1. **To Divide Platform Binaries**: Add a condition to the compilation file (lib.sh).
3. **Modify Firmware Version** in the appropriate platform file.
4. **Integrate QN Program** into the build.
5. **Integrate Our Services** in the build (apply to each platform).
6. **Modify SSH Service Port Number**
7. **Remove Telnet Service** from the build.
8. **Add Standalone GUI** in Build.
9. **Replace rc.conf**
10. **Replace 1**
11. **Add true in  sw-config.json**
12. **Add `ip ssh password-auth` in  default_config.txt**

---


### 1) To Divide Platform Binaries

**Why is this needed in the firmware?**

We support multiple platforms and every platform has different-different architecture. The architecture is different so SDK and compilation tools or processes are different, so generated binaries are also different, it means Platform-specific binaries do not support other systems. So every platform should have supported binaries. That's why we need to platform wise binaries. 

**Steps:**

Add a condition in the `lib.sh` file for dividing platform binaries:

- Edit the `lib_install_packages` function in the "src" directory.
- Add the necessary conditions between the `mkdir` and `cp` commands.

``` bash
    if [ $PLATFORM == 'QN-CAT3' ]; then
        cp -r ${TOP_DIR}/QN_PKGS_AC3/build/*         ./rootfs/usr/bin/
    elif [ $PLATFORM == 'QN-A2' ]; then
        cp -r ${TOP_DIR}/QN_PKGS_A2/build/*         ./rootfs/usr/bin/
    elif [ $PLATFORM == 'QN-ROS7' ]; then
        cp -r ${TOP_DIR}/QN_PKGS_AC5X/build/*       ./rootfs/usr/bin/
    fi
```

For a visual comparison of the changes, refer to this link: [Diff Checker](https://www.diffchecker.com/7FXQ7xy2/)

---
---

### 2) Modify Firmware Version

**Why is this needed in the firmware?**

For the same identity for every firmware, that’s why we keep the same version pattern for every firmware. 


**Steps:**

- Open the `build-qn-sw225-cat3.sh` , `build-qn-a2.sh` and `build-qn-ros7.sh`   file in the “src” directory.
- Modify the `VERSION` variable by adding “00” behind its value.

Example:
- **Old:** `VERSION=’XX.XX.XX’`
- **New:** `VERSION=’XX.XX.XX.00’`

- Add below lines after `SDK_DIR=${TOP_DIR}/sdk-qn-sw225-cat3` , `SDK_DIR=${TOP_DIR}/sdk-qn-a2` and `SDK_DIR=${TOP_DIR}/sdk-qn-ros7`
``` shell
    echo "${VERSION}" | tee ./fw_version
    cp ./fw_version ./rootfs/common/etc/
```
For a visual comparison of the changes, refer to this link: [Diff Checker](https://www.diffchecker.com/sMhsoeTH/)


---
---

### 3) Integrate QN Program into the Build

**Why is this needed?**

The QN Program includes binaries and scripts to achieve certain goals and targets.

**Steps:**

- Clone or download the program from [GitHub](https://github.com/Khimji07/Quantum_switch/tree/main/Firmware_build_use_only).
- Add the downloaded folders `QN_PKGS_A2/build`, `QN_PKGS_AC3/build` and `QN_PKGS_AC5X/build` to the earlier extracted SDK.

---
---

### 4) Integrate Our Services in the Build

**Why is this needed in the firmware?**

A service is used to schedule some task or perform the same task repeatedly. In achieving some goals and repeatedly completing tasks, we use our own services. Follow the below steps to add services to our quantum firmware. 

**Steps:**

- Clone or download the services from [GitHub](https://github.com/Khimji07/Quantum_switch/tree/main/Firmware_build_use_only/Services).
- Add the services (`QNTM`) to the extracted SDK.
- Execute the shell script files in each service folder separately(`qntm.sh`).

---
---

### 5) Modify SSH Service Port Number

**Why is this needed in the firmware?**

For security reasons, we need to modify our SSH service port instead of its default. Follow the below steps to modify SSH service port. 

**Steps:**

- Open the `run` file located in `/extracted_SDK_Path/rootfs/common/etc/sv/dropbear/`.
- Modify the port number from `2222` to `22222`.

```shell
    #!/bin/sh -u
    exec 2>&1
    . /etc/runit/functions

    msg 'starting...'

    msg 'executing dropbear binary...'
    exec dropbear -R -F -E -p 22222
```
For a visual comparison of the changes, refer to this link: [Diff Checker](https://www.diffchecker.com/iwioSzo2/)

---
---

### 6) Remove Telnet Service

**Why is this needed in the firmware?**

One of the main risks of using Telnet over public networks is that it does not encrypt any of the data that is transmitted between the client and the server. That’s why we need to remove linux telnet service from quantum firmware. 


**Steps:**

- Remove the `telnetd` directory from `/extracted_SDK_Path/rootfs/common/etc/sv/`.
- Also, remove the `telnetd` link file from `/extracted_SDK_Path/rootfs/common/etc/runit/runsvdir/default/`.

---
---

### 7) Add Standalone GUI in Build

**Why is this needed in the firmware?**

A standalone GUI is a switch-independent GUI. Using the standalone GUI, we can manage the switch GUI locally (within the network). 

**Steps:**

- Clone or download the “www” directory from [GitHub](https://github.com/Khimji07/Quantum_switch/tree/main/Firmware_build_use_only).
- Compress the www folder to `webdevice-qntmnet-9999-1.xpak.tar.bz2` using the command: `tar -cjvf webdevice-qntmnet-9999-1.xpak.tar.bz2 www`.
- Copy it to two different locations in the extracted SDK.
    1) /extrected_SDK_path/replica/output/cat3/webdevice-qntmnet-9999-1.xpak.tar.bz2
    2) /extrected_SDK_path/replica/output/a2/webdevice-qntmnet-9999-1.xpak.tar.bz2
    3) /extrected_SDK_path/replica/output/ros7/webdevice-qntmnet-9999-1.xpak.tar.bz2

---
---

### 8) Replace rc.conf

- Download the `rc.conf` file from [GitHub](https://github.com/Khimji07/Quantum_switch/tree/main/Firmware_build_use_only).
- Replace it in `/extracted_SDK_Path/rootfs/cat3/etc/rc.conf`.

For a visual comparison of the changes, refer to this link: [Diff Checker](https://www.diffchecker.com/iwioSzo2/)

- Download the `rc.conf` file from [GitHub](https://github.com/Khimji07/Quantum_switch/tree/main/Firmware_build_use_only).
- Replace it in `/extracted_SDK_Path/rootfs/ros7/etc/rc.conf`.

For a visual comparison of the changes, refer to this link: [Diff Checker](https://www.diffchecker.com/iwioSzo2/)

---
---

### 9) Replace 1

- Download the `1` file from [GitHub](https://github.com/Khimji07/Quantum_switch/tree/main/Firmware_build_use_only).
- Replace it in `/extracted_SDK_Path/rootfs/common/etc/runit/1`.

For a visual comparison of the changes, refer to this link: [Diff Checker](https://www.diffchecker.com/iwioSzo2/)

---
---

### 10) Add true in  sw-config.json

- make change in `sw-config.json` file located in `/extracted_SDK_Path/rootfs/common/etc/skel/sw-config.json`
- change `"gui_toggle": false` to  `"gui_toggle": true`

For a visual comparison of the changes, refer to this link: [Diff Checker](https://www.diffchecker.com/iwioSzo2/)

---
---


### 11) Add `ip ssh password-auth` in  default_config.txt

- make change in `default_config.txt` file located in `/extracted_SDK_Path/rootfs/common/ros/current/default_config.txt`
- add `ip ssh password-auth` below  `ip ssh server`

For a visual comparison of the changes, refer to this link: [Diff Checker](https://www.diffchecker.com/iwioSzo2/)

---
---

## Building Process

To compile the firmware, follow these steps for individual switches:

### For Alleycat3

- Navigate to the extracted SDK folder.
- Execute the command: `./src/build-qn-sw225-ac3.sh`.

### For Aldrin2

- Navigate to the extracted SDK folder.
- Execute the command: `./src/build-qn-sw225-a2.sh`.

### For Alleycat5 and Data  Center switch

- Navigate to the extracted SDK folder.
- Execute the command: `./src/build-qn-sw225-ros7.sh`.
---

