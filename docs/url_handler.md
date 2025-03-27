# URL Handler

## Purpose
The URL Handler script is designed to modify the `rudder_url` in `sw-config.json` to facilitate the pre-provisioning of network switches to the cloud.

## Full Forms
- **NSC**: Network Switch Controller
- **QNR**: QN Live Rudder URL → [https://rudder.qntmnet.com](https://rudder.qntmnet.com)
- **QNRUS**: QN US Rudder URL → [https://us.rudder.qntmnet.com](https://us.rudder.qntmnet.com)
- **QNRUAT**: QN UAT Rudder URL → [https://rudder.uat.qntmnet.com](https://rudder.uat.qntmnet.com)
- **QNRDEV**: QN Dev Rudder URL → [https://rudder.dev.qntmnet.com](https://rudder.dev.qntmnet.com)

## How to Use
To update the `rudder_url`, run the following command in the CLI of the switch:

- To change to US Rudder URL:
  ```sh
  copy nsc qnrus
  ```
- To change to Dev Rudder URL:
  ```sh
  copy nsc qnrdev
  ```
- To change to UAT Rudder URL:
  ```sh
  copy nsc qnruat
  ```
- To change to Live Rudder URL:
  ```sh
  copy nsc qnr
  ```

## Things to Note
- This script **will not work** if the switch is already onboarded.
- The script **only works** when the switch is in **Quick Setup Mode**.
- Ensure the `rudder_url` is correctly updated in `sw-config.json` using the `more sw-config.json` command to verify changes.

