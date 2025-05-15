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
            <prefix>1131.108.5.0</prefix>
            <mask>255.255.255.255</mask>
            <next-hop>131.108.1.12</next-hop>
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

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <routing xmlns="yang:set_route">
        <static-routes>
          <route operation="delete">
            <prefix>131.108.5.0</prefix>
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

Parameters:

```
null
```

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

Define an ip address for an interface.

Parameters:

| Name | Value Type | Requirement | Description |
| ---- | ---------- | ----------- | ----------- |
| `name` | String | mandatory | Interfaces are: gi1/0/1, gi1/0/2, ..., gi1/0/48, te1/0/1, te1/0/2, te1/0/3, te1/0/4. |
| `ip_address` | String | mandatory | Specifies the IP address. |
| `mask_prefix` | String | mandatory | Network mask of the IP address or prefix length: The number of bits that comprise the IP address prefix. |

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

### Enable/Disable Ports, Set Description and Speed Ports
Turns up/down a set port that has been turned off (does nothing if the port isn't in use).
Configures the administrative status, adds a descriptive label, and sets the speed for a port.

Parameters:

| Name | Value Type | Requirement | Description |
| ---- | ---------- | ----------- | ----------- |
| `name` | String | mandatory | The port name, valid values are: gi1/0/1 - gi1/0/48 for Ethernet, te1/0/1 - te1/0/4 for SFP, Po1 - Po32 for port channel. |
| `admin_status` | String | optional | The admin-status, valid values are: up or down|
| `description` | String | optional | Specifies a comment or a description of the port to assist the user. |
| `speed` | Integer | optional | The set speed, valid values are: 10, 100, 1000, 10000. |

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <port-configurations xmlns="yang:set_port_config">
        <port>
          <name>te1/0/1</name>
          <admin-status>down</admin-status>
          <description>PC</description>
          <speed>10000</speed>
        </port>
        <port>
          <name>te1/0/2</name>
          <description>Co_Switch</description>
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
A port in access mode can be an untagged member of maximally a single VLAN.

Parameters:

| Name | Value Type | Requirement | Description |
| ---- | ---------- | ----------- | ----------- |
| `name` | String | mandatory | Ports are: gi1/0/1, gi1/0/2, ..., gi1/0/48, te1/0/1, te1/0/2, te1/0/3, te1/0/4, Po1, Po2, ..., Po32. |
| `vlan_id` | String | mandatory | Specifies the VLAN to which the port is configured. Set this argument to "none" to specify that the access port cannot belong to any VLAN. |

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <port-configurations xmlns="yang:set_port_config">
        <port>
          <name>te1/0/6</name>
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
Configure the VLAN membership mode.

Parameters:

| Name | Value Type | Requirement | Description |
| ---- | ---------- | ----------- | ----------- |
| `name` | String | mandatory | Specify the interface name (One of: gi1/0/1, gi1/0/2, ..., gi1/0/48, te1/0/1, te1/0/2, te1/0/3, te1/0/4) or range of interfaces with range interface_name. See examples. |
| `mode` | String | mandatory | The argument can take one of the following values: access: Specifies an untagged layer 2 VLAN port; trunk: Specifies a trunking layer 2 VLAN port; general: Specifies a full 802-1q-supported VLAN port; private-vlan promiscuous: Private-VLAN promiscuous port; private-vlan host: Private-VLAN host port; customer: Specifies that an edge port connected to customer equipment. Traffic received from this port will be tunneled with the additional 802.1q. VLAN tag(Q-in-Q VLANtunneling). |

#### Request
```xml
<rpc>
  <edit-config>
    <config>
      <port-configurations xmlns="yang:set_port_config">
        <port>
          <name>te1/0/10</name>
          <switchport>
            <mode>trunk</mode>
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

---

## 7. Spanning Tree Protocol (STP) Global
Spanning Tree Protocol prevents broadcast storms and loop issues in a switched network. This section covers global STP settings.


### Get Global STP Status
Checks if STP is globally enabled or disabled on the device.(true - enable, false - disable)

Parameters:

```
null
```
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
Enable spanning tree functionality.

Parameters:

```
null
```

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
Disable spanning tree functionality.

Parameters:

```
null
```

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

The desired state of the interface (1 = UP, 2 = DOWN, 3 = TESTING).

Parameters:

| Name | Value Type | Description |
| ---- | ---------- | ----------- |
| `interface-number` | Integer | Interface number. |

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

Get interface description.

Parameters:

| Name | Value Type | Description |
| ---- | ---------- | ----------- |
| `interface-number` | Integer | Interface number. |

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

An estimate of the interface's current bandwidth in bits per second.

Parameters:

| Name | Value Type | Description |
| ---- | ---------- | ----------- |
| `interface-number` | Integer | Interface number. |

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