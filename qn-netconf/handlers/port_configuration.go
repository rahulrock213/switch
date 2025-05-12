package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"strconv"
	"strings"

	"qn-netconf/miyagi"
)

const PortConfigNamespace = "urn:example:params:xml:ns:yang:port-config"
const NetconfBaseNamespacePortConfig = "urn:ietf:params:xml:ns:netconf:base:1.0"

// --- Common NETCONF XML Data Structures ---
type RpcReplyPortConfig struct {
	XMLName   xml.Name             `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
	MessageID string               `xml:"message-id,attr"`
	Data      *DataPortConfig      `xml:"data,omitempty"`
	Ok        *OkPortConfig        `xml:"ok,omitempty"`
	Errors    []RPCErrorPortConfig `xml:"rpc-error,omitempty"`
}

type OkPortConfig struct {
	XMLName xml.Name `xml:"ok"`
}

type DataPortConfig struct {
	XMLName            xml.Name            `xml:"data"`
	PortConfigurations *PortConfigurations `xml:"port-configurations,omitempty"`
}

type RPCErrorPortConfig struct {
	XMLName       xml.Name `xml:"rpc-error"`
	ErrorType     string   `xml:"error-type"`
	ErrorTag      string   `xml:"error-tag"`
	ErrorSeverity string   `xml:"error-severity"`
	ErrorMessage  string   `xml:"error-message"`
}

// --- Port Configuration Specific XML Data Structures ---

type PortConfigurations struct {
	XMLName xml.Name          `xml:"port-configurations"`
	Xmlns   string            `xml:"xmlns,attr,omitempty"`
	Ports   []PortConfigEntry `xml:"port"`
}

type PortConfigEntry struct {
	XMLName     xml.Name          `xml:"port"`
	Operation   string            `xml:"operation,attr,omitempty"` // For edit-config
	Name        string            `xml:"name"`
	AdminStatus *string           `xml:"admin-status,omitempty"` // "up" or "down"
	Speed       *string           `xml:"speed,omitempty"`        // e.g., "1000", "auto" - Miyagi takes int for set
	Description *string           `xml:"description,omitempty"`
	Switchport  *SwitchportConfig `xml:"switchport,omitempty"`
	// Stp          *StpConfig        `xml:"stp,omitempty"` // STP removed for now
}

type SwitchportConfig struct {
	XMLName    xml.Name        `xml:"switchport"`
	Mode       *string         `xml:"mode,omitempty"` // "access" or "trunk"
	AccessVlan *AccessVlanInfo `xml:"access,omitempty"`
	TrunkVlan  *TrunkVlanInfo  `xml:"trunk,omitempty"`
}

type AccessVlanInfo struct {
	XMLName xml.Name `xml:"access"`
	VlanID  *int     `xml:"vlan-id,omitempty"`
}

type TrunkVlanInfo struct {
	XMLName      xml.Name `xml:"trunk"`
	AllowedVlans *string  `xml:"allowed-vlans,omitempty"` // "10,20,30-35"
	NativeVlanID *int     `xml:"native-vlan-id,omitempty"`
}

// STP removed for now
// type StpConfig struct {
// 	XMLName xml.Name `xml:"stp"`
// 	Enabled *bool    `xml:"enabled,omitempty"`
// }

// --- Handler Functions ---

// HandlePortConfigurationEditConfig handles <edit-config> for port configurations
func HandlePortConfigurationEditConfig(miyagiSocketPath string, request []byte, msgID, frameEnd string) []byte {
	var editReq struct {
		XMLName            xml.Name           `xml:"config"`
		PortConfigurations PortConfigurations `xml:"port-configurations"`
	}

	configStartIndex := bytes.Index(request, []byte("<config>"))
	configEndIndex := bytes.LastIndex(request, []byte("</config>"))
	if configStartIndex == -1 || configEndIndex == -1 || configStartIndex >= configEndIndex {
		return buildErrorResponseBytesPortConfig(msgID, "protocol", "malformed-message", "Malformed <edit-config> request", frameEnd)
	}
	configPayload := request[configStartIndex : configEndIndex+len("</config>")]

	if err := xml.Unmarshal(configPayload, &editReq); err != nil {
		log.Printf("NETCONF_PORT_CFG_HANDLER: Error unmarshalling <edit-config> payload: %v. Payload: %s", err, string(configPayload))
		return buildErrorResponseBytesPortConfig(msgID, "protocol", "malformed-message", "Invalid port configuration format", frameEnd)
	}

	if len(editReq.PortConfigurations.Ports) == 0 {
		return buildErrorResponseBytesPortConfig(msgID, "protocol", "missing-element", "<port> element is required under <port-configurations>", frameEnd)
	}

	for _, portCfg := range editReq.PortConfigurations.Ports {
		if portCfg.Name == "" {
			return buildErrorResponseBytesPortConfig(msgID, "protocol", "missing-element", "Port <name> is required", frameEnd)
		}

		// 1. Admin Status (Enable/Disable Port)
		if portCfg.AdminStatus != nil {
			var miyagiUID string
			if strings.ToLower(*portCfg.AdminStatus) == "up" || strings.ToLower(*portCfg.AdminStatus) == "enable" {
				miyagiUID = "Agent.Switch.Set.Port.On"
			} else if strings.ToLower(*portCfg.AdminStatus) == "down" || strings.ToLower(*portCfg.AdminStatus) == "disable" {
				miyagiUID = "Agent.Switch.Set.Port.Off"
			} else {
				return buildErrorResponseBytesPortConfig(msgID, "protocol", "bad-attribute", "Invalid admin-status value. Use 'up' or 'down'.", frameEnd)
			}
			err := callMiyagiPortConfig(miyagiSocketPath, miyagiUID, map[string]interface{}{"name": portCfg.Name}, msgID, portCfg.Name, "admin-status")
			if err != nil {
				return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
			}
		}

		// 2. Speed
		if portCfg.Speed != nil {
			speedVal, err := strconv.Atoi(*portCfg.Speed) // Assuming Miyagi takes int
			if err != nil {
				// Handle "auto" or other string values if Miyagi supports them differently
				return buildErrorResponseBytesPortConfig(msgID, "protocol", "invalid-value", fmt.Sprintf("Invalid speed value: %s. Expected integer.", *portCfg.Speed), frameEnd)
			}
			err = callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.Port.Speed", map[string]interface{}{"name": portCfg.Name, "speed": speedVal}, msgID, portCfg.Name, "speed")
			if err != nil {
				return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
			}
		}

		// 3. Description
		if portCfg.Description != nil {
			err := callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.Port.InterfaceDescription", map[string]interface{}{"interface_name": portCfg.Name, "string": *portCfg.Description}, msgID, portCfg.Name, "description")
			if err != nil {
				return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
			}
		}

		// 4. Switchport Configuration
		if portCfg.Switchport != nil {
			// Mode (Trunk/Access)
			if portCfg.Switchport.Mode != nil {
				mode := strings.ToLower(*portCfg.Switchport.Mode)
				if mode != "access" && mode != "trunk" {
					return buildErrorResponseBytesPortConfig(msgID, "protocol", "invalid-value", "Switchport mode must be 'access' or 'trunk'.", frameEnd)
				}
				err := callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.Trunking.VlanMembershipMode", map[string]interface{}{"interface_name": portCfg.Name, "mode": mode}, msgID, portCfg.Name, "switchport mode")
				if err != nil {
					return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
				}
			}

			// Access VLAN
			if portCfg.Switchport.AccessVlan != nil && portCfg.Switchport.AccessVlan.VlanID != nil {
				if portCfg.Switchport.Mode == nil || strings.ToLower(*portCfg.Switchport.Mode) != "access" {
					log.Printf("NETCONF_PORT_CFG_HANDLER: Setting access VLAN for %s. Ensure mode is 'access' or will be set by agent.", portCfg.Name)
				}
				err := callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.VLAN.SwitchportAccess", map[string]interface{}{"interface_name": portCfg.Name, "vlan_id": strconv.Itoa(*portCfg.Switchport.AccessVlan.VlanID)}, msgID, portCfg.Name, "access vlan")
				if err != nil {
					return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
				}
			}

			// Trunk VLANs
			if portCfg.Switchport.TrunkVlan != nil {
				args := map[string]interface{}{"interface_name": portCfg.Name}
				changed := false
				if portCfg.Switchport.TrunkVlan.AllowedVlans != nil {
					args["vlan_id"] = *portCfg.Switchport.TrunkVlan.AllowedVlans
					changed = true
				}
				if portCfg.Switchport.TrunkVlan.NativeVlanID != nil {
					args["native_vlan_id"] = *portCfg.Switchport.TrunkVlan.NativeVlanID
					changed = true
				}
				if changed {
					if portCfg.Switchport.Mode == nil || strings.ToLower(*portCfg.Switchport.Mode) != "trunk" {
						log.Printf("NETCONF_PORT_CFG_HANDLER: Setting trunk VLANs for %s. Ensure mode is 'trunk' or will be set by agent.", portCfg.Name)
					}
					err := callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.Trunking.SwitchportTrunkAndNativeVlan", args, msgID, portCfg.Name, "trunk vlans")
					if err != nil {
						return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
					}
				}
			}
		}
		// STP Configuration removed
	}

	reply := RpcReplyPortConfig{MessageID: msgID, Ok: &OkPortConfig{}}
	return marshalToXMLPortConfig(reply, frameEnd)
}

// HandlePortConfigurationGetConfig handles <get-config> for port configurations
func HandlePortConfigurationGetConfig(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiAllIntReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{"uid": "Agent.Switch.Get.General.AllInterfaces", "arg": nil},
		ID:     11, // Unique ID
	}

	miyagiAllIntResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiAllIntReq)
	if err != nil || miyagiAllIntResp.Error != nil {
		errMsg := "Error communicating with device agent for AllInterfaces"
		if err != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, err)
		} else {
			errMsg = fmt.Sprintf("Device error for AllInterfaces: %s (code: %d)", miyagiAllIntResp.Error.Message, miyagiAllIntResp.Error.Code)
		}
		log.Printf("NETCONF_PORT_CFG_HANDLER: %s", errMsg)
		return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	var miyagiInterfaceMap map[string]MiyagiInterfaceDetail // From interface.go
	if err := json.Unmarshal(miyagiAllIntResp.Result, &miyagiInterfaceMap); err != nil {
		log.Printf("NETCONF_PORT_CFG_HANDLER: Error unmarshalling AllInterfaces: %v. Raw: %s", err, string(miyagiAllIntResp.Result))
		return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", "Failed to parse interface data from device", frameEnd)
	}

	var portConfigEntries []PortConfigEntry
	for name, details := range miyagiInterfaceMap {
		entry := PortConfigEntry{Name: name}

		// Admin Status
		if details.IfAdminStatus != nil {
			if details.IfAdminStatus.Value != nil {
				if *details.IfAdminStatus.Value == 1 { // Assuming 1 is Up
					status := "up"
					entry.AdminStatus = &status
				} else {
					status := "down"
					entry.AdminStatus = &status
				}
			}
		}

		// Speed
		// Note: Agent.Switch.Get.Interface.Speed might be needed for more accurate current speed
		// Agent.Switch.Get.General.AllInterfaces might provide configured or operational speed.
		// For now, using IfSpeed from AllInterfaces.
		if details.IfSpeed != nil {
			speedStr := strconv.Itoa(*details.IfSpeed) // Assuming IfSpeed is in Mbps
			entry.Speed = &speedStr
		}

		// Description
		// Note: Agent.Switch.Get.Interface.Description might be needed if IfDescription is not sufficient.
		if details.IfDescription != "" {
			entry.Description = &details.IfDescription
		}

		// Switchport Info
		swConfig := SwitchportConfig{}
		changedSw := false
		if details.PortMode != nil && details.PortMode.Description != "" {
			mode := strings.ToLower(details.PortMode.Description) // "access" or "trunk"
			swConfig.Mode = &mode
			changedSw = true

			if mode == "access" && len(details.UntaggedVlan) > 0 {
				vlanID := details.UntaggedVlan[0]
				swConfig.AccessVlan = &AccessVlanInfo{VlanID: &vlanID}
			} else if mode == "trunk" {
				trunkInfo := TrunkVlanInfo{}
				trunkChanged := false
				if len(details.TaggedVlan) > 0 {
					var vlanStrings []string
					for _, v := range details.TaggedVlan {
						vlanStrings = append(vlanStrings, strconv.Itoa(v))
					}
					allowed := strings.Join(vlanStrings, ",") // Consider if Miyagi returns ranges or just list
					trunkInfo.AllowedVlans = &allowed
					trunkChanged = true
				}
				if details.NativeVlan != nil {
					trunkInfo.NativeVlanID = details.NativeVlan
					trunkChanged = true
				}
				if trunkChanged {
					swConfig.TrunkVlan = &trunkInfo
				}
			}
		}
		if changedSw {
			entry.Switchport = &swConfig
		}

		// STP Status get logic removed

		portConfigEntries = append(portConfigEntries, entry)
	}

	portConfigurations := PortConfigurations{
		Xmlns: PortConfigNamespace,
		Ports: portConfigEntries,
	}

	reply := RpcReplyPortConfig{
		MessageID: msgID,
		Data: &DataPortConfig{
			PortConfigurations: &portConfigurations,
		},
	}
	return marshalToXMLPortConfig(reply, frameEnd)
}

// Helper to make Miyagi calls and handle common error logic
func callMiyagiPortConfig(miyagiSocketPath, uid string, args map[string]interface{}, msgID, portName, configItem string) error {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{"uid": uid, "arg": args},
		ID:     12, // Consider a unique ID generator or incrementing counter
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_PORT_CFG_HANDLER: Error calling Miyagi for %s on port %s (%s): %v", uid, portName, configItem, err)
		return fmt.Errorf("Error communicating with device agent for %s on port %s", configItem, portName)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error for %s on port %s (%s): %s (code: %d)", configItem, portName, uid, miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_PORT_CFG_HANDLER: Miyagi returned error: %s", errMsg)
		return fmt.Errorf(errMsg)
	}
	log.Printf("NETCONF_PORT_CFG_HANDLER: Successfully applied %s for port %s using UID %s", configItem, portName, uid)
	return nil
}

// --- Helper Functions ---
func marshalToXMLPortConfig(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_PORT_CFG_HANDLER: FATAL: Failed to marshal XML: %v", err)
		return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rpc-reply xmlns="%s"><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`, NetconfBaseNamespacePortConfig, frameEnd))
	}
	return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
}

func buildErrorResponseBytesPortConfig(msgID, errType, errTag, errMsg, frameEnd string) []byte {
	escapedErrMsg := strings.ReplaceAll(errMsg, "<", "&lt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, ">", "&gt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, "&", "&amp;")
	reply := RpcReplyPortConfig{
		MessageID: msgID,
		Errors:    []RPCErrorPortConfig{{ErrorType: errType, ErrorTag: errTag, ErrorSeverity: "error", ErrorMessage: escapedErrMsg}},
	}
	return marshalToXMLPortConfig(reply, frameEnd)
}

// Note: MiyagiInterfaceDetail struct is defined in interface.go.
// It's assumed to be accessible here as they are in the same 'handlers' package.
