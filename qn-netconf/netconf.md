# NETCONF Usage Guide 

## APIs
- VLAN
- SSH
- TELNET
- Routing
- IP Interface
- Port Configuration
- STP Global
- Get Port Status
- Get Port Description
- Get Port Speed
---

## Logging into NETCONF Client via SSH

To start a NETCONF session over SSH from a Linux system, use the following command:

```bash
ssh -p 830 <username>@<device_ip> -s netconf
```

**Example:**

```bash
ssh -p 830 admin@192.168.1.1 -s netconf
```

This command:

* Connects to the network device using SSH.
* Initiates a NETCONF session (`-s netconf` is required to switch into NETCONF subsystem).

Ensure the target device has NETCONF enabled and listens on port 830 (default for NETCONF over SSH).

---

## 1. VLAN Configuration
Virtual Local Area Networks (VLANs) allow you to segment your network. This section shows how to view and manage VLANs.

### Get VLANs
Retrieves a list of all configured VLANs and their names. 

Parameters:

```
null
```

#### Request
```xml
<rpc>
  <get>
    <vlans xmlns="yang:vlan"/>
  </get>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply xmlns="yang:vlan">
  <vlans xmlns="yang:get_vlan">
    <vlan>
      <id>1</id>
      <name>Default</name>
    </vlan>
    <vlan>
      <id>2</id>
      <name>Account</name>
    </vlan>
    <vlan>
      <id>3</id>
      <name>HR</name>
    </vlan>
  </vlans>
</rpc-reply>
]]>]]>
```

### Set VLAN
Creates a new VLAN or modifies an existing one.

Parameters:

| Name | Value Type | Requirement | Description |
| ---- | ---------- | ----------- | ----------- |
| `id` | Integer | mandatory | The id of the vlan. |
| `name` | String | optional | The name of the vlan (if not entered, id will be used as name). |

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <vlans xmlns="yang:set_vlan">
        <vlan>  
          <id>100</id>
          <name>vlan_100</name>
        </vlan> 
      </vlans>
    </config>
  </edit-config>
</rpc>
]]>]]>
```


#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply xmlns="yang:vlan">
  <result>ok</result>
</rpc-reply>
]]>]]>
```
---

## 2. SSH Server Configuration
Secure Shell (SSH) provides secure remote access to the device.

### Get SSH Status
Checks if the SSH server is currently enabled or disabled. (true - enable, false - disable)

Parameters:

```
null
```

#### Request
```xml
<rpc>
  <get>
    <ssh xmlns="yang:get_ssh"/>
  </get>
</rpc>
]]>]]>
```


#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <ssh>
    <enabled>true</enabled>
  </ssh>
</rpc-reply>
]]>]]>
```

### Enable SSH
Turns on the SSH server.

Parameters:

```
null
```

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <ssh xmlns="yang:set_ssh">
        <enabled>true</enabled>
      </ssh>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

### Disable SSH
Turns off the SSH server.

Parameters:

```
null
```

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <ssh xmlns="yang:set_ssh">
        <enabled>false</enabled>
      </ssh>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

---

## 3. Telnet Server Configuration
Telnet provides remote access, but it's less secure than SSH as data is sent in clear text.



### Get Telnet Status
Checks if the Telnet server is currently enabled or disabled.(true = enable telnet, false = disable telnet)

Parameters:

```
null
```

#### Request
```xml
<rpc>
  <get>
    <telnet xmlns="yang:get_telnet"/>
  </get>
 </rpc>
 ]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <telnet>
    <enabled>false</enabled>
  </telnet>
</rpc-reply>
]]>]]>
```

### Enable Telnet
Turns on the Telnet server.

Parameters:

```
null
```

#### Request
```xml
<rpc >
  <edit-config>
    <config>
      <telnet xmlns="yang:set_telnet">
        <enabled>true</enabled>
      </telnet>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

### Disable Telnet
Turns off the Telnet server.

Parameters:

```
null
```
#### Request
```xml
<rpc >
  <edit-config>
    <config>
      <telnet xmlns="yang:set_telnet">
        <enabled>false</enabled>
      </telnet>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

---

## 4. Static Routing
Static routes manually define paths for IP traffic.

### Add Static Route
Adds a new static IP route to the device's routing table. The `operation="create"` attribute indicates that a new route entry should be created.

Parameters:

| Name | Value Type | Requirement | Description |
| ---- | ---------- | ----------- | ----------- |
| `prefix` | String | mandatory | The IP route prefix for the destination. |
| `mask` | String | mandatory | The mask for the destination. |
| `next-hop` | String | mandatory | IP address of the next hop that can be used to reach the network. |

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <routing xmlns="yang:set_route">
        <static-routes>
          <route operation="create">
            <prefix>131.108.1.27</prefix>
            <mask>255.255.255.255</mask>
            <next-hop>131.108.1.28</next-hop>
          </route>
        </static-routes>
      </routing>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

### Delete Static Route
Removes an existing static IP route. The `operation="delete"` attribute specifies that the matching route entry should be removed.

Parameters:

| Name | Value Type | Requirement | Description |
| ---- | ---------- | ----------- | ----------- |
| `prefix` | String | mandatory | The IP route prefix for the destination. |
| `mask` | String | mandatory | The mask for the destination. |
| `next_hop` | String | mandatory | IP address of the next hop that can be used to reach the network. |

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <routing xmlns="yang:set_route">
        <static-routes>
          <route operation="delete">
            <prefix>131.108.1.27</prefix>
            <mask>255.255.255.255</mask>
          </route>
        </static-routes>
      </routing>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

---

## 5. IP Interface
This section deals with configuring IP addresses and subnet masks on network interfaces.


### Get All IP Interfaces
Retrieves the IP address configuration for all interfaces that have an IP address assigned.

#### Request
```xml
<rpc>
  <get>
      <ip-interfaces xmlns="yang:get_ip_interface"/>
  </get>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <ip-interfaces>
    <1>
      <ip4>172.16.100.163</ip4>
      <subnet_mask>255.255.255.0</subnet_mask>
      <type>
        <value>2</value>
        <description>dhcp</description>
      </type>
      <ifindex>100000</ifindex>
    </1>
    <MTPLAP_point_to_point_Port_1>
      <ip4>203.0.113.121</ip4>
      <subnet_mask>255.255.255.0</subnet_mask>
      <type>
        <value>1</value>
        <description>static</description>
      </type>
      <ifindex>8000</ifindex>
    </MTPLAP_point_to_point_Port_1>
    <oob>
      <ip4>192.168.254.254</ip4>
      <subnet_mask>255.255.255.0</subnet_mask>
      <type>
        <value>1</value>
        <description>static</description>
      </type>
      <ifindex>1080</ifindex>
    </oob>
  </ip-interfaces>
</rpc-reply>
]]>]]>
```

### Set IP Interface
Assigns an IP address and subnet mask to a specified interface. The `operation="create"` attribute is used here to define a new IP address configuration on the interface. If an IP configuration already exists, this might update it or add a secondary address depending on the device's behavior (often, `merge` or `replace` operations are used for updates).

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <ip-interfaces xmlns="yang:set_ip_interface">
        <interface>
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

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

---

## 6. Port Configuration
This section covers various settings for physical switch ports, such as administrative status, speed, description, and VLAN membership modes (access/trunk).

### Enable/Disable Ports
Sets the administrative status of specified ports to 'up' (enabled) or 'down' (disabled).

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <port-configurations xmlns="yang:set_port_config">
        <port><name>te1/0/1</name><admin-status>up</admin-status></port>
        <port><name>te1/0/2</name><admin-status>up</admin-status></port>
      </port-configurations>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

### Set Description and Speed
Configures the administrative status, adds a descriptive label, and sets the speed for a port.

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <port-configurations xmlns="yang:set_port_config">
        <port>
          <name>te1/0/1</name>
          <admin-status>up</admin-status>
          <description>PC</description>
          <speed>10000</speed>
        </port>
        <port>
          <name>te1/0/2</name>
          <admin-status>up</admin-status>
          <description>Core_Switch</description>
        </port>
      </port-configurations>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

### Configure Access VLAN
Sets a port to 'access' mode and assigns it to a specific VLAN. Packets on an access port are untagged and belong to this single VLAN.

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <port-configurations xmlns="yang:set_port_config">
        <port>
          <name>te1/0/1</name>
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

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

### Configure Trunk VLANs
Sets a port to 'trunk' mode, allowing it to carry traffic for multiple VLANs. You can specify which VLANs are allowed and set a native VLAN (for untagged traffic).

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <port-configurations xmlns="yang:set_port_config">
        <port>
          <name>te1/0/5</name>
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

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

### Enable/Disable STP on Port
Configures Spanning Tree Protocol (STP) on a specific port. This is typically for per-port STP settings if the device supports it, distinct from global STP. (Note: This example assumes per-port STP control. Global STP is covered in the next section.)

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <port-configurations xmlns="yang:set_port_config">
        <port>
          <name>te1/0/6</name>
          <stp><enabled>true</enabled></stp>
        </port>
      </port-configurations>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

---

## 7. Spanning Tree Protocol (STP) Global
Spanning Tree Protocol prevents broadcast storms and loop issues in a switched network. This section covers global STP settings.


### Get Global STP Status
Checks if STP is globally enabled or disabled on the device.

#### Request
```xml
<rpc>
  <get>
    <stp-global-config xmlns="yang:get_stp"/>
  </get>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <stp-global-config>
    <enabled>true</enabled>
  </stp-global-config>
</rpc-reply>
]]>]]>
```

### Enable Global STP
Turns on STP for the entire device.

#### Request
```xml
<rpc>
  <edit-config>
    <target><running/></target>
    <config>
      <stp-global-config xmlns="yang:set_stp">
        <enabled>true</enabled>
      </stp-global-config>
    </config>
  </edit-config>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```

### Disable Global STP
Turns off STP for the entire device.

#### Request
```xml
<rpc>
  <edit-config>
    <target><running/></target>
    <config>
      <stp-global-config xmlns="yang:set_stp">
        <enabled>false</enabled>
      </stp-global-config>
    </config>
  </edit-config>
</rpc>
]]>]]>
```
#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <result>ok</result>
</rpc-reply>
]]>]]>
```
---

## 8. Get Port Status

To retrieve the operational status of a specific port by its interface number.

#### Request
```xml
<rpc>
  <get>
      <port-status xmlns="yang:get_port_status">
        <interface-number>1</interface-number>
      </port-status>
  </get>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <data>
    <port-status xmlns="yang:get_port_status">
      <interface-number>1</interface-number>
      <status>
        <value>1</value>
        <description>UP</description>
      </status>
    </port-status>
  </data>
</rpc-reply>
]]>]]>
```

---

## 9. Get Port Description

To retrieve the configured description of a specific port by its interface number.

#### Request
```xml
<rpc>
  <get>
    <port-description xmlns="yang:get_port_description">
      <interface-number>1</interface-number>
    </port-description>
  </get>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <data>
    <port-description xmlns="yang:get_port_description">
      <interface-number>1</interface-number>
      <description>PC</description>
    </port-description>
  </data>
</rpc-reply>
]]>]]>
```

---

## 11. Get Port Speed

To retrieve the operational speed of a specific port by its interface number.

#### Request
```xml
<rpc>
  <get>
    <port-speed xmlns="yang:get_port_speed">
      <interface-number>1</interface-number>
    </port-speed>
  </get>
</rpc>
]]>]]>
```

#### Response
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply>
  <data>
    <port-speed xmlns="yang:get_port_speed">
      <interface-number>1</interface-number>
      <speed>10000</speed>
    </port-speed>
  </data>
</rpc-reply>
]]>]]>
```