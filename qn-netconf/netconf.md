# NETCONF Usage Guide for Interface, VLAN, SSH, Telnet, Routing, IP Interface, Ports, and STP

This guide provides NETCONF XML payload examples for managing a network device via its YANG model. These examples include retrieving and editing configurations related to interfaces, VLANs, SSH/Telnet settings, static routes, IP interfaces, and spanning tree protocol (STP).

---

## 1. Interface Configuration

### Get Interface Information

```xml
<rpc message-id="102" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <filter type="subtree">
      <interfaces xmlns="urn:example:params:xml:ns:yang:interfaces"/>
    </filter>
  </get>
</rpc>
```

---

## 2. VLAN Configuration

### Get VLANs

```xml
<rpc message-id="102" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <vlans xmlns="urn:example:params:xml:ns:yang:vlan"/>
  </get>
</rpc>
```

### Set VLAN

```xml
<rpc message-id="101" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><running/></target>
    <config>
      <vlans xmlns="urn:example:params:xml:ns:yang:vlan">
        <vlan>
          <id>19</id>
          <name>vlan19</name>
        </vlan>
      </vlans>
    </config>
  </edit-config>
</rpc>
```

---

## 3. SSH Server Configuration

### Get SSH Status

```xml
<rpc message-id="120" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get>
    <filter type="subtree">
      <ssh-server-config xmlns="urn:example:params:xml:ns:yang:ssh-server-config"/>
    </filter>
  </get>
</rpc>
```

### Enable SSH

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
```

### Disable SSH

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
```

---

## 4. Telnet Server Configuration

### Get Telnet Status

```xml
<rpc message-id="130" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source><running/></source>
    <filter type="subtree">
      <telnet-server-config xmlns="urn:example:params:xml:ns:yang:telnet-server-config"/>
    </filter>
  </get-config>
</rpc>
```

### Enable Telnet

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
```

### Disable Telnet

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
```

---

## 5. Static Routing

### Add Static Route

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
```

### Delete Static Route

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
```

---

## 6. IP Interface

### Get All IP Interfaces

```xml
<rpc message-id="150" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source><running/></source>
    <filter type="subtree">
      <ip-interfaces xmlns="urn:example:params:xml:ns:yang:ip-interface"/>
    </filter>
  </get-config>
</rpc>
```

### Set IP Interface

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
```

---

## 7. Port Configuration

### Enable/Disable Ports

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
```

### Set Description and Speed

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
```

### Configure Access VLAN

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
```

### Configure Trunk VLANs

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
```

### Enable/Disable STP on Port

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
```

---

## 8. Spanning Tree Protocol (STP) Global

### Get Global STP Status

```xml
<rpc message-id="170" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source><running/></source>
    <filter type="subtree">
      <stp-global-config xmlns="urn:example:params:xml:ns:yang:stp-global-config"/>
    </filter>
  </get-config>
</rpc>
```

### Enable Global STP

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
```

### Disable Global STP

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
```

---

## Notes

* Replace interface names, VLAN IDs, IP addresses, etc., with your device's values.
* Always confirm YANG module support using the device's capabilities (`<hello>` message).
* `edit-config` modifies the running datastore; `get` and `get-config` fetch state and configuration.

---

For more on NETCONF and YANG:

* [RFC 6241 - NETCONF](https://datatracker.ietf.org/doc/html/rfc6241)
* [YANG Language Tutorial](https://tools.ietf.org/html/rfc6020)
