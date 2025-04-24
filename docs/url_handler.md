# URL Handler (Dynamic)

## Purpose  
The URL Handler script is designed to modify the `rudder_url` in `sw-config.json` to facilitate the pre-provisioning of QN switches to **custom Rudder endpoints**.

## How to Use  
To update the `rudder_url`, run one of the following commands in the CLI of the switch:

- To use a custom HTTP Rudder URL:
  ```sh
  copy nsc http-<ip>
  ```
- To use a custom HTTPS Rudder URL:
  ```sh
  copy nsc https-<ip>
  ```

### Examples:
- For HTTP:
  ```sh
  copy nsc http-172.16.100.29
  ```
- For HTTPS:
  ```sh
  copy nsc https-172.16.100.29
  ```

> These commands create a control file in `/mnt/flash` that the script uses to convert into a full Rudder URL.

## Things to Note  
- This script **will not work** if the switch is already onboarded.  
- The script **only works** when the switch is in **Quick Setup Mode**.  
- The file name must be in the format `http-<ip>` or `https-<ip>`.  
- After execution, the `rudder_url` in `sw-config.json` and the `url5` entry in the preprovision service config are updated.  
- Use `more sw-config.json` to verify the URL change.
