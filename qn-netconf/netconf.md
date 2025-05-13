# NETCONF Usage Guide for Interface, VLAN, SSH, Telnet, Routing, IP Interface, Ports, and STP

This guide provides practical NETCONF XML examples to help you manage and configure your network device. NETCONF (Network Configuration Protocol) is a standardized protocol for managing network devices, using XML-based data encoding for configuration and operational data.

These examples cover common tasks such as:
*   Retrieving current configurations and operational state.
*   Modifying device settings for interfaces, VLANs, remote access (SSH/Telnet), IP routing, IP addressing, port-specific features, and Spanning Tree Protocol (STP).

**Key NETCONF Concepts Used in This Guide:**

*   **`<rpc>`:** The root element for every NETCONF request. It includes a `message-id` attribute, which is a string chosen by the client to uniquely identify the request. The server will use the same `message-id` in its response.
*   **`<get>`:** Retrieves operational state data and configuration data.
*   **`<get-config>`:** Retrieves configuration data. You specify the datastore to retrieve from (e.g., `<running/>` for the active configuration).
*   **`<edit-config>`:** Modifies configuration data. You specify the target datastore (e.g., `<running/>`) and provide the configuration changes within a `<config>` element.
*   **`<filter type="subtree">`:** Used with `<get>` or `<get-config>` to specify which parts of the configuration or state data you want to retrieve. You provide an XML structure representing the desired data.
*   **`<config>`:** Used within `<edit-config>` to enclose the configuration data you want to apply.
*   **`xmlns` (XML Namespace):** An attribute used to qualify XML elements and attributes, preventing naming conflicts. Each data model (e.g., for VLANs, interfaces) will have its own namespace.
*   **`operation` attribute:** Used within `<edit-config>` on specific data nodes to indicate the action to perform (e.g., `create`, `delete`, `merge`, `replace`).

---

## 1. Interface Configuration
This section covers how to retrieve detailed information about all network interfaces on the device.

### Get Interface Information
Retrieves operational status and configuration for all interfaces.

```xml
<rpc>
  <get>
    <interfaces xmlns="yang:get_interface"/>
  </get>
</rpc>
]]>]]>
```

---

## 2. VLAN Configuration
Virtual Local Area Networks (VLANs) allow you to segment your network. This section shows how to view and manage VLANs.

### Get VLANs
Retrieves a list of all configured VLANs and their names. The response format for this specific `<get>` operation is a custom XML structure, where VLAN data is directly under a `<vlans>` element.

```xml
<rpc>
  <get>
    <vlans xmlns="yang:get_vlan"/>
  </get>
</rpc>
]]>]]>
```

### Set VLAN
Creates a new VLAN or modifies an existing one.


```xml
<rpc>
  <edit-config>
    <config>
      <vlans xmlns="yang:set_vlan">
        <vlan>  
          <id>79</id>
          <name>vlan_79</name>
        </vlan> 
      </vlans>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

---

## 3. SSH Server Configuration
Secure Shell (SSH) provides secure remote access to the device.


### Get SSH Status
Checks if the SSH server is currently enabled or disabled.


```xml
<rpc message-id="120" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <filter type="subtree">
      <ssh-server-config xmlns="urn:example:params:xml:ns:yang:ssh-server-config"/>
    </filter>
  </get>
</rpc>
]]>]]>
```

### Enable SSH
Turns on the SSH server.


```xml
<rpc message-id="121" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <ssh-server-config xmlns="urn:example:params:xml:ns:yang:ssh-server-config">
        <enabled>true</enabled>
      </ssh-server-config>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

### Disable SSH
Turns off the SSH server.


```xml
<rpc message-id="122" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <ssh-server-config xmlns="urn:example:params:xml:ns:yang:ssh-server-config">
        <enabled>false</enabled>
      </ssh-server-config>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

---

## 4. Telnet Server Configuration
Telnet provides remote access, but it's less secure than SSH as data is sent in clear text.


### Get Telnet Status
Checks if the Telnet server is currently enabled or disabled.


```xml
<rpc message-id="130" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source><running/></source>
    <filter type="subtree">
      <telnet-server-config xmlns="urn:example:params:xml:ns:yang:telnet-server-config"/>
    </filter>
  </get-config>
</rpc>
]]>]]>
```

### Enable Telnet
Turns on the Telnet server.


```xml
<rpc message-id="131" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <telnet-server-config xmlns="urn:example:params:xml:ns:yang:telnet-server-config">
        <enabled>true</enabled>
      </telnet-server-config>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

### Disable Telnet
Turns off the Telnet server.


```xml
<rpc message-id="132" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <telnet-server-config xmlns="urn:example:params:xml:ns:yang:telnet-server-config">
        <enabled>false</enabled>
      </telnet-server-config>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

---

## 5. Static Routing
Static routes manually define paths for IP traffic.


### Add Static Route
Adds a new static IP route to the device's routing table. The `operation="create"` attribute indicates that a new route entry should be created.


```xml
<rpc message-id="140" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <routing xmlns="urn:example:params:xml:ns:yang:routing">
        <static-routes>
          <route operation="create">
            <prefix>192.168.100.15</prefix>
            <mask>255.255.255.255</mask>
            <next-hop>192.168.100.1</next-hop>
          </route>
        </static-routes>
      </routing>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

### Delete Static Route
Removes an existing static IP route. The `operation="delete"` attribute specifies that the matching route entry should be removed.


```xml
<rpc message-id="141" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <routing xmlns="urn:example:params:xml:ns:yang:routing">
        <static-routes>
          <route operation="delete">
            <prefix>192.168.100.1</prefix>
            <mask>255.255.255.255</mask>
          </route>
        </static-routes>
      </routing>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

---

## 6. IP Interface
This section deals with configuring IP addresses and subnet masks on network interfaces.


### Get All IP Interfaces
Retrieves the IP address configuration for all interfaces that have an IP address assigned.


```xml
<rpc message-id="150" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source><running/></source>
    <filter type="subtree">
      <ip-interfaces xmlns="urn:example:params:xml:ns:yang:ip-interface"/>
    </filter>
  </get-config>
</rpc>
]]>]]>
```

### Set IP Interface
Assigns an IP address and subnet mask to a specified interface. The `operation="create"` attribute is used here to define a new IP address configuration on the interface. If an IP configuration already exists, this might update it or add a secondary address depending on the device's behavior (often, `merge` or `replace` operations are used for updates).

```xml
<rpc message-id="151" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <ip-interfaces xmlns="urn:example:params:xml:ns:yang:ip-interface">
        <interface operation="create">
          <name>te1/0/1</name>
          <ip-address>131.108.1.27</ip-address>
          <mask-prefix>255.255.255.0</mask-prefix>
        </interface>
      </ip-interfaces>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

---

## 7. Port Configuration
This section covers various settings for physical switch ports, such as administrative status, speed, description, and VLAN membership modes (access/trunk).

### Enable/Disable Ports
Sets the administrative status of specified ports to 'up' (enabled) or 'down' (disabled).


```xml
<rpc message-id="161" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <port-configurations xmlns="urn:example:params:xml:ns:yang:port-config">
        <port><name>te1/0/1</name><admin-status>up</admin-status></port>
        <port><name>te1/0/2</name><admin-status>up</admin-status></port>
      </port-configurations>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

### Set Description and Speed
Configures the administrative status, adds a descriptive label, and sets the speed for a port.


```xml
<rpc message-id="161" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <port-configurations xmlns="urn:example:params:xml:ns:yang:port-config">
        <port>
          <name>te1/0/1</name>
          <admin-status>up</admin-status>
          <description>Uplink_to_Core_Switch</description>
          <speed>10000</speed>
        </port>
        <port>
          <name>te1/0/2</name>
          <admin-status>down</admin-status>
        </port>
      </port-configurations>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

### Configure Access VLAN
Sets a port to 'access' mode and assigns it to a specific VLAN. Packets on an access port are untagged and belong to this single VLAN.


```xml
<rpc message-id="162" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <port-configurations xmlns="urn:example:params:xml:ns:yang:port-config">
        <port>
          <name>te1/0/3</name>
          <switchport>
            <mode>access</mode>
            <access><vlan-id>100</vlan-id></access>
          </switchport>
        </port>
      </port-configurations>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

### Configure Trunk VLANs
Sets a port to 'trunk' mode, allowing it to carry traffic for multiple VLANs. You can specify which VLANs are allowed and set a native VLAN (for untagged traffic).


```xml
<rpc message-id="163" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <port-configurations xmlns="urn:example:params:xml:ns:yang:port-config">
        <port>
          <name>te1/0/4</name>
          <switchport>
            <mode>trunk</mode>
            <trunk>
              <allowed-vlans>10,20,30-35,40</allowed-vlans>
              <native-vlan-id>1</native-vlan-id>
            </trunk>
          </switchport>
        </port>
      </port-configurations>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

### Enable/Disable STP on Port
Configures Spanning Tree Protocol (STP) on a specific port. This is typically for per-port STP settings if the device supports it, distinct from global STP. (Note: This example assumes per-port STP control. Global STP is covered in the next section.)


```xml
<rpc message-id="164" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <port-configurations xmlns="urn:example:params:xml:ns:yang:port-config">
        <port>
          <name>te1/0/5</name>
          <stp><enabled>true</enabled></stp>
        </port>
      </port-configurations>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

---

## 8. Spanning Tree Protocol (STP) Global
Spanning Tree Protocol prevents broadcast storms and loop issues in a switched network. This section covers global STP settings.


### Get Global STP Status
Checks if STP is globally enabled or disabled on the device.


```xml
<rpc message-id="170" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source><running/></source>
    <filter type="subtree">
      <stp-global-config xmlns="urn:example:params:xml:ns:yang:stp-global-config"/>
    </filter>
  </get-config>
</rpc>
]]>]]>
```

### Enable Global STP
Turns on STP for the entire device.


```xml
<rpc message-id="171" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <stp-global-config xmlns="urn:example:params:xml:ns:yang:stp-global-config">
        <enabled>true</enabled>
      </stp-global-config>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

### Disable Global STP
Turns off STP for the entire device.


```xml
<rpc message-id="172" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <stp-global-config xmlns="urn:example:params:xml:ns:yang:stp-global-config">
        <enabled>false</enabled>
      </stp-global-config>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

---

## Important Notes

* **Placeholders**: The XML examples use placeholder values for interface names (e.g., `te1/0/1`), VLAN IDs (e.g., `19`), IP addresses, etc. You must replace these with the actual values relevant to your device and desired configuration.
* **Message ID (`message-id`):** While the examples use sequential message IDs (e.g., `101`, `102`), you can use any unique string. The server will echo this ID in its response, helping you match requests with replies.
* **Device Capabilities**: Before attempting these operations, it's good practice to check the device's capabilities, which are advertised in its initial NETCONF `<hello>` message. This tells you which NETCONF features and YANG data models the device supports.
* **Datastores**: <edit-config> typically targets the `<running/>` datastore (the active configuration). Other datastores like `<candidate/>` (for a staging area before committing) might be supported depending on the device.
* **Error Handling**: If a NETCONF operation fails, the server will respond with an `<rpc-error>` element containing details about the error.

---

For more on NETCONF and YANG:

* [RFC 6241 - NETCONF](https://datatracker.ietf.org/doc/html/rfc6241)
* [YANG Language Tutorial](https://tools.ietf.org/html/rfc6020)
