# SFTP Handler

## Purpose

The **SFTP Handler** enables SFTP access between a switch and an external SFTP server (such as a PC) for uploading and downloading configuration files and firmware images. The switch CLI provides simple `copy` commands, which generate specially named trigger files that are monitored and executed by the `sftp.sh` script.

---

## Command Formats

### 1. ✅ Enable SFTP to a Target Device

Use this command to establish SFTP access to a switch using its IP, port, username, and password.

**Command Format:**
```sh
copy sftp_enable ip-<ip_addr>_port-<port>_user-<username>_pass-<password>
```

**Example:**
```sh
copy sftp_enable ip-172.16.100.29_port-8000_user-newadmin_pass-1234
```

---

## ⚠️ Path Format Note

> **Note:**  
> In all upload and download commands, use a **dot (`.`)** in place of the **slash (`/`)** to represent directory paths.  
> For example:  
> - `path-uploads.startup-config` → remote path `uploads/startup-config`  
> - `path-backups.configs.myfile.cfg` → remote path `backups/configs/myfile.cfg`

---

## Upload Commands (Switch ➜ PC/SFTP Server)

### 2. 📤 Upload Startup Configuration

Upload the current **startup-config** from the switch **to the SFTP server**.

**Command Format:**
```sh
copy sftp_upload_startup path-<path>
```

**Example:**
```sh
copy sftp_upload_startup path-uploads.startup-config
```

---

### 3. 📤 Upload Running Configuration

Upload the current **running-config** from the switch **to the SFTP server**.

**Command Format:**
```sh
copy sftp_upload_running path-<path>
```

**Example:**
```sh
copy sftp_upload_running path-uploads.running-config
```

---

## Download Commands (PC/SFTP Server ➜ Switch)

### 4. 📥 Download Startup Configuration

Download a **startup-config** file from the SFTP server **to the switch**.

**Command Format:**
```sh
copy sftp_download_startup path-<path>
```

**Example:**
```sh
copy sftp_download_startup path-uploads.startup-config
```

---

### 5. 📥 Download Running Configuration

Download a **running-config** file from the SFTP server **to the switch**.

**Command Format:**
```sh
copy sftp_download_running path-<path>
```

**Example:**
```sh
copy sftp_download_running path-uploads.running-config
```

---

### 6. 💾 Download Image File

Download a **firmware image file** from the SFTP server **to the switch**.

**Command Format:**
```sh
copy sftp_download_image path-<path>
```

**Example:**
```sh
copy sftp_download_image path-uploads.image_QN-ROS7-2.3.8.02.qntm
```

---

## Summary

| Action                        | Direction         | Command Example                                                  |
|------------------------------|-------------------|------------------------------------------------------------------|
| Enable SFTP                  | —                 | `copy sftp_enable ip-172.16.100.29_port-8000_user-admin_pass-123` |
| Upload startup-config        | Switch ➜ Server   | `copy sftp_upload_startup path-uploads.startup-config`           |
| Download startup-config      | Server ➜ Switch   | `copy sftp_download_startup path-uploads.startup-config`         |
| Upload running-config        | Switch ➜ Server   | `copy sftp_upload_running path-uploads.running-config`           |
| Download running-config      | Server ➜ Switch   | `copy sftp_download_running path-uploads.running-config`         |
| Download image               | Server ➜ Switch   | `copy sftp_download_image path-uploads.image_<firmware>.qntm`    |
