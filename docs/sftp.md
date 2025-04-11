# SFTP Handler

## Purpose
The **SFTP Handler** allows you to enable SFTP access, upload configurations to, and download configurations from an SFTP server — all via simple `copy` commands in the CLI. These commands generate specially named files that trigger actions monitored by the `sftp.sh` script.

---

## Command Formats

### 1. ✅ Enable SFTP to a Target Device

Use this command to trigger SFTP access to a switch using its IP, port, username, and password.

**Command Format:**
```sh
copy sftp_enable ip-<ip_addr>_port-<port>_user-<username>_pass-<password>
```

**Example:**
```sh
copy sftp_enable ip-172.16.100.29_port-8000_user-newadmin_pass-1234
```

---

### 2. 📤 Upload Configuration to SFTP Server

Use this to upload configuration or data from the switch to the server.

**Command Format:**
```sh
copy sftp_upload path-<path>
```

**Example:**
```sh
copy sftp_upload path-uploads
```


---

### 3. 📥 Download Configuration from SFTP Server

Use this command to download a configuration file from the SFTP server to the switch.

**Command Format:**
```sh
copy sftp_download path-<path>
```

> **Note:**  
> In the filename, use a **dot (`.`)** to represent **directory separators (`/`)** in the remote path.  
> For example, `path-uploads.startup-config` corresponds to the remote path `uploads/startup-config`.

**Example:**
```sh
copy sftp_download path-uploads.startup-config
```
