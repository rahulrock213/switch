# MP-BGP MPLS Topology 

This diagram shows the MPLS core with MP-BGP running between P and PE routers, connecting two customer sites (AS-1 and AS-2).


```mermaid
graph TD
  %% AS 100 Routers
  subgraph AS100[AS 100]
    R1[200-2<br>3.3.3.3/32]
    R2[400-1<br>2.2.2.2/32]
    R3[200-1<br>4.4.4.4/32]
  end

  %% AS 1 Routers and Client
  subgraph AS1[AS-1]
    R4[100-1<br>1.1.1.1/32]
    C1[C-5<br>10.100.5.2/24]
  end

  %% AS 2 Routers and Client
  subgraph AS2[AS-2]
    R5[100-2<br>5.5.5.5/32]
    C2[C-6<br>10.100.6.2/24]
  end

  %% Core Links
  R1 -->|10.10.20.0/24| R3
  R1 -->|10.10.36.0/24| R2

  %% AS-1 Links
  R2 -->|10.10.12.0/24| R4
  R4 -->|10.100.5.0/24| C1

  %% AS-2 Links
  R3 -->|10.10.14.0/24| R5
  R5 -->|10.100.6.0/24| C2
  
```mermaid

## R1 - QN-SR-100-1


```bash

# IP Assignment
admin@QN-SR-100-1:~$ sudo config loopback add Loopback1
admin@QN-SR-100-1:~$ sudo config interface ip add Ethernet4 10.10.12.1/24
admin@QN-SR-100-1:~$ sudo config interface ip add Ethernet5 10.100.5.1/24
admin@QN-SR-100-1:~$ sudo config interface ip add Loopback1 1.1.1.1/32

# eBGP Configurations
QN-SR-100-1# configure 
QN-SR-100-1(config)# router bgp 1
QN-SR-100-1(config-router)# bgp router-id 1.1.1.1
QN-SR-100-1(config-router)# neighbor 10.10.12.2 remote-as 100
QN-SR-100-1(config-router)# network 10.100.5.0/24


```

## R2 - QN-SR-400-1

```bash

# IP Assignment
admin@QN-SR-400-1:~$ sudo config loopback add Loopback2
admin@QN-SR-400-1:~$ sudo config interface ip add Ethernet1 10.10.12.2/24
admin@QN-SR-400-1:~$ sudo config interface ip add Ethernet0 10.10.23.2/24
admin@QN-SR-400-1:~$ sudo config interface ip add Loopback2 2.2.2.2/32

# Vrf Configuration
admin@QN-SR-400-1:~$ sudo config vrf add Vrfcustomer1
admin@QN-SR-400-1:~$ sudo config interface vrf bind Ethernet1 Vrfcustomer1
admin@QN-SR-400-1:~$ sudo config interface ip add Ethernet1 10.10.12.2/24

# OSPF Configurations
admin@QN-SR-400-1:~$ vtysh
QN-SR-400-1# configure terminal  
QN-SR-400-1(config)# router ospf 
QN-SR-400-1(config-router)# ospf router-id 2.2.2.2
QN-SR-400-1(config-router)# network 10.10.23.0/24 area 0
QN-SR-400-1(config-router)# network 2.2.2.2/32 area 0

# LDP Configurations
admin@QN-SR-400-1:~$ sudo config interface mpls add Ethernet0
QN-SR-400-1(config)# mpls ldp 
QN-SR-400-1(config-ldp)# router-id 2.2.2.2
QN-SR-400-1(config-ldp)# address-family ipv4
QN-SR-400-1(config-ldp-af)# discovery transport-address 2.2.2.2
QN-SR-400-1(config-ldp-af)# interface Ethernet0
QN-SR-400-1(config-ldp-af-if)# exit
QN-SR-400-1(config-ldp-af)# exit-address-family

# MP-iBGP Configurations
QN-SR-400-1# configure terminal 
QN-SR-400-1(config)# router bgp 100
QN-SR-400-1(config-router)# bgp router-id 2.2.2.2
QN-SR-400-1(config-router)# neighbor 4.4.4.4 remote-as 100
QN-SR-400-1(config-router)# neighbor 4.4.4.4 update-source Loopback2
QN-SR-400-1(config-router)# address-family ipv4 vpn
QN-SR-400-1(config-router-af)# neighbor 4.4.4.4 activate 
QN-SR-400-1(config-router-af)# exit-address-family 
QN-SR-400-1(config-router)# address-family ipv4 unicast 
QN-SR-400-1(config-router-af)# no neighbor 4.4.4.4 activate 
QN-SR-400-1(config-router-af)# exit-address-family

# BGP VRF Instance Configurations
QN-SR-400-1(config)# router bgp 100 vrf Vrfcustomer1 
QN-SR-400-1(config-router)# no bgp ebgp-requires-policy 
QN-SR-400-1(config-router)# neighbor 10.10.12.1 remote-as 1
QN-SR-400-1(config-router)# address-family ipv4 unicast 
QN-SR-400-1(config-router-af)# redistribute connected 
QN-SR-400-1(config-router-af)# label vpn export auto 
QN-SR-400-1(config-router-af)# rd vpn export 100:1
QN-SR-400-1(config-router-af)# rt vpn both 100:1
QN-SR-400-1(config-router-af)# export vpn 
QN-SR-400-1(config-router-af)# import vpn

```
**NOTE** 
- When you add interface in vrf, you'll have to add the IP address again.
- Once the interface has been added, ensure it is selected when verifying connectivity.<br> Example: ```ping -I <Interface_name> <Destination> ```

## R3 - QN-SR-200-2

```bash

# IP Assignment
admin@QN-SR-200-2:~$ sudo config loopback add Loopback3
admin@QN-SR-200-2:~$ sudo config interface ip add Ethernet0 10.10.23.3/24
admin@QN-SR-200-2:~$ sudo config interface ip add Ethernet1 10.10.34.3/24
admin@QN-SR-200-2:~$ sudo config interface ip add Loopback3 3.3.3.3/32

# OSPF Configurations
admin@QN-SR-200-2:~$ vtysh 
QN-SR-200-2# configure terminal 
QN-SR-200-2(config)# router ospf
QN-SR-200-2(config-router)# ospf router-id 3.3.3.3
QN-SR-200-2(config-router)# network 10.10.23.0/24 area 0
QN-SR-200-2(config-router)# network 10.10.34.0/24 area 0
QN-SR-200-2(config-router)# network 3.3.3.3/32 area 0

# LDP Configurations
admin@QN-SR-200-2:~$ sudo config interface mpls add Ethernet0
admin@QN-SR-200-2:~$ sudo config interface mpls add Ethernet1
QN-SR-200-2(config)# mpls ldp                 
QN-SR-200-2(config-ldp)# router-id 3.3.3.3
QN-SR-200-2(config-ldp)# address-family ipv4 
QN-SR-200-2(config-ldp-af)# discovery transport-address 3.3.3.3
QN-SR-200-2(config-ldp-af)# interface Ethernet0
QN-SR-200-2(config-ldp-af-if)# exit
QN-SR-200-2(config-ldp-af)# interface Ethernet1
QN-SR-200-2(config-ldp-af-if)# exit
QN-SR-200-2(config-ldp-af)# exit-address-family
```

## R4 - QN-SR-200-1

```bash

# IP Assignment
admin@QN-SR-200-1:~$ sudo config loopback add Loopback4
admin@QN-SR-200-1:~$ sudo config interface ip add Ethernet1 10.10.34.4/24
admin@QN-SR-200-1:~$ sudo config interface ip add Ethernet0 10.10.45.4/24
admin@QN-SR-200-1:~$ sudo config interface ip add Loopback4 4.4.4.4/32

# Vrf Configuration
admin@QN-SR-200-1:~$ sudo config vrf add Vrfcustomer1
admin@QN-SR-200-1:~$ sudo config interface vrf bind Ethernet0 Vrfcustomer1
admin@QN-SR-200-1:~$ sudo config interface ip add Ethernet0 10.10.45.4/24

# OSPF Configurations
admin@QN-SR-200-1:~$ sudo config interface mpls add Ethernet1
admin@QN-SR-200-1:~$ vtysh 
QN-SR-200-1# configure terminal
QN-SR-200-1(config)# router ospf
QN-SR-200-1(config-router)# ospf router-id 4.4.4.4
QN-SR-200-1(config-router)# network 10.10.34.0/24 area 0
QN-SR-200-1(config-router)# network 4.4.4.4/32 area 0

# LDP Configurations
QN-SR-200-1(config)# mpls ldp 
QN-SR-200-1(config-ldp)# router-id 4.4.4.4
QN-SR-200-1(config-ldp)# address-family ipv4 
QN-SR-200-1(config-ldp-af)# discovery transport-address 4.4.4.4 
QN-SR-200-1(config-ldp-af)# interface Ethernet1
QN-SR-200-1(config-ldp-af-if)# exit
QN-SR-200-1(config-ldp-af)# exit-address-family

# MP-iBGP Configurations
QN-SR-200-1(config)# router bgp 100
QN-SR-200-1(config-router)# bgp router-id 4.4.4.4
QN-SR-200-1(config-router)# neighbor 2.2.2.2 remote-as 100
QN-SR-200-1(config-router)# neighbor 2.2.2.2 update-source Loopback4
QN-SR-200-1(config-router)# address-family ipv4 vpn 
QN-SR-200-1(config-router-af)# neighbor 2.2.2.2 activate 
QN-SR-200-1(config-router-af)# exit-address-family 
QN-SR-200-1(config-router)# address-family ipv4 unicast 
QN-SR-200-1(config-router-af)# no neighbor 2.2.2.2 activate 
QN-SR-200-1(config-router-af)# exit-address-family 

# BGP VRF Instance Configurations
QN-SR-200-1(config)# router bgp 100 vrf Vrfcustomer1 
QN-SR-200-1(config-router)# no bgp ebgp-requires-policy 
QN-SR-200-1(config-router)# neighbor 10.10.45.5 remote-as 2
QN-SR-200-1(config-router)# address-family ipv4 unicast 
QN-SR-200-1(config-router-af)# redistribute connected 
QN-SR-200-1(config-router-af)# label vpn export auto 
QN-SR-200-1(config-router-af)# rd vpn export 100:1
QN-SR-200-1(config-router-af)# rt vpn export 100:1
QN-SR-200-1(config-router-af)# export vpn 
QN-SR-200-1(config-router-af)# import vpn 
QN-SR-200-1(config-router-af)# exit-address-family

```
**NOTE** When you add interface in vrf, you'll have to add the IP address again.
- Once the interface has been added, ensure it is selected when verifying connectivity.<br> Example: ```ping -I <Interface_name> <Destination> ```


## R5 - QN-SR-50-1

```bash

# IP Assignment
admin@QN-SR-200-2:~$ sudo config loopback add Loopback5
admin@QN-SR-200-2:~$ sudo config interface ip add Ethernet4 10.10.45.5/24
admin@QN-SR-200-2:~$ sudo config interface ip add Ethernet5 10.10.6.1/24
admin@QN-SR-200-2:~$ sudo config interface ip add Loopback5 5.5.5.5/32

# eBGP Configurations
admin@QN-SR-50-1:~$ vtysh 
QN-SR-50-1# configure terminal 
QN-SR-50-1(config)# router bgp 2
QN-SR-50-1(config-router)# no bgp ebgp-requires-policy 
QN-SR-50-1(config-router)# neighbor 10.10.45.4 remote-as 100
QN-SR-50-1(config-router)# network 10.100.6.0/24

```
