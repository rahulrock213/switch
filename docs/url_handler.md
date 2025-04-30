# URL Handler (Dynamic)

## Purpose  
The **URL Handler** updates the `rudder_url` in `sw-config.json`, allowing the switch to connect to a **custom Rudder server** (HTTP or HTTPS).  
This is used during **Quick Setup Mode** for pre-provisioning.

---

## How to Use  

Use the `copy nsc` command in the switch CLI with either HTTP or HTTPS, and optionally include a port.

### 🔸 HTTP Rudder URL
```sh
copy nsc http-<ip>[_<port>]
```

**Examples:**
```sh
copy nsc http-172.16.100.29
copy nsc http-172.16.100.29_8000
```

---

### 🔸 HTTPS Rudder URL
```sh
copy nsc https-<ip>[_<port>]
```

**Examples:**
```sh
copy nsc https-172.16.100.29
copy nsc https-172.16.100.29_8443
```

> These commands create a file in `/mnt/flash` that the script reads to update the Rudder URL.

---

## Notes

- Works **only** in **Quick Setup Mode**
- Does **not** work if the switch is already onboarded
- Port is **optional**  
- The `rudder_url` in `sw-config.json` and `url5` in the preprovision config are updated
- You can check the updated URL using:
  ```sh
  more sw-config.json
  ```
