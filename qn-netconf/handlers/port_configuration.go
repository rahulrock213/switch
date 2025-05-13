package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"qn-netconf/miyagi"
)

const PortConfigNamespace = "urn:example:params:xml:ns:yang:port-config"
const NetconfBaseNamespacePortConfig = "urn:ietf:params:xml:ns:netconf:base:1.0"

// --- Common NETCONF XML Data Structures ---
type RpcReplyPortConfig struct {
	// Simplified for generic <ok/>, <error>, and custom GET data
	XMLName xml.Name `xml:"rpc-reply"`
	// MessageID removed
	// Data field removed for custom GETs; specific data structs will be embedded or marshalled directly
	Ok     *OkPortConfig        `xml:"ok,omitempty"`
	Errors []RPCErrorPortConfig `xml:"rpc-error,omitempty"` // Used for edit-config errors
	// For custom GET responses, specific data structs will be added here if needed,
	// or the response will be built as a raw XML string.
}

// RpcReplyPortConfigGetData is specifically for the GET physical port configurations response
type RpcReplyPortConfigGetData struct {
	XMLName xml.Name        `xml:"rpc-reply"` // Simplified root
	Data    *DataPortConfig `xml:"data,omitempty"`
}

type OkPortConfig struct {
	XMLName xml.Name `xml:"ok"`
}

type DataPortConfig struct { // This is used by the original HandlePortConfigurationGetConfig for physical ports
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

// --- Physical Port Configuration Specific XML Data Structures ---

type PortConfigurations struct {
	// XMLName made namespace-agnostic for edit-config unmarshalling.
	XMLName xml.Name          `xml:"port-configurations"` // For edit-config unmarshalling
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

// --- Port Channel (LAG) Specific XML Data Structures ---

// For GET Port Channels Response
type LagGetResponseRoot struct {
	XMLName      xml.Name              `xml:"rpc-reply"`
	PortChannels []LagEntryGetResponse `xml:"port-channel"`
}

type LagEntryGetResponse struct {
	XMLName  xml.Name `xml:"port-channel"`
	Name     string   `xml:"name"`
	LacpMode string   `xml:"lacp-mode,omitempty"`
	Members  *Members `xml:"members,omitempty"`
	// Status    string   `xml:"status,omitempty"` // Optional: if Miyagi provides it
}

type Members struct {
	Member []string `xml:"member"`
}

// For Edit-Config Port Channels Request
type EditConfigLagPayload struct {
	XMLName           xml.Name              `xml:"config"`
	LagConfigurations LagConfigurationsEdit `xml:"port-channels"`
}

type LagConfigurationsEdit struct {
	XMLName      xml.Name       `xml:"port-channels"` // Namespace will be on this tag in request
	PortChannels []LagEntryEdit `xml:"port-channel"`
}

type LagEntryEdit struct {
	XMLName   xml.Name        `xml:"port-channel"`
	Operation string          `xml:"operation,attr"` // create, delete, modify
	Name      string          `xml:"name"`
	LacpMode  *string         `xml:"lacp-mode,omitempty"` // active, passive, on
	Members   *LagMembersEdit `xml:"members,omitempty"`
}

type LagMembersEdit struct {
	Member []LagMemberEdit `xml:"member"`
}

type LagMemberEdit struct {
	XMLName   xml.Name `xml:"member"`
	Operation string   `xml:"operation,attr,omitempty"` // add, remove
	Value     string   `xml:",chardata"`
}

// Miyagi specific structs for Port Channels
type MiyagiLagDetail struct {
	Members  []string `json:"members"`
	LacpMode string   `json:"lacp_mode"`
	// Status string `json:"status"` // if available
}

// --- Handler Functions ---

// HandlePortConfigurationEditConfig handles <edit-config> for physical port configurations
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

	reply := RpcReplyPortConfig{
		// MessageID removed
		Ok: &OkPortConfig{},
	}
	return marshalToXMLPortConfig(reply, frameEnd)
}

// HandlePortConfigurationGetConfig handles <get-config> for physical port configurations
// This function returns a standard NETCONF reply with <data> wrapper.
func HandlePortConfigurationGetConfig(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiAllIntReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{"uid": "Agent.Switch.Get.General.AllInterfaces", "arg": nil},
		ID:     11, // Unique ID
	}

	miyagiAllIntResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiAllIntReq)
	if err != nil || (miyagiAllIntResp != nil && miyagiAllIntResp.Error != nil) {
		errMsg := "Error communicating with device agent for AllInterfaces"
		if err != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, err)
		} else if miyagiAllIntResp.Error != nil {
			errMsg = fmt.Sprintf("Device error for AllInterfaces: %s (code: %d)", miyagiAllIntResp.Error.Message, miyagiAllIntResp.Error.Code)
		}
		log.Printf("NETCONF_PORT_CFG_HANDLER: %s", errMsg)
		// Use a RpcReplyPortConfig that includes MessageID for standard replies
		errorReply := struct { // Local struct for standard error reply
			XMLName   xml.Name             `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
			MessageID string               `xml:"message-id,attr"`
			Errors    []RPCErrorPortConfig `xml:"rpc-error,omitempty"`
		}{
			MessageID: msgID,
			Errors:    []RPCErrorPortConfig{{ErrorType: "application", ErrorTag: "operation-failed", ErrorSeverity: "error", ErrorMessage: errMsg}},
		}
		xmlBytes, _ := xml.MarshalIndent(errorReply, "", "  ")
		return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
	}

	var miyagiInterfaceMap map[string]MiyagiInterfaceDetail // From interface.go
	if err := json.Unmarshal(miyagiAllIntResp.Result, &miyagiInterfaceMap); err != nil {
		log.Printf("NETCONF_PORT_CFG_HANDLER: Error unmarshalling AllInterfaces: %v. Raw: %s", err, string(miyagiAllIntResp.Result))
		// Use a RpcReplyPortConfig that includes MessageID for standard replies
		errorReply := struct {
			XMLName   xml.Name             `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
			MessageID string               `xml:"message-id,attr"`
			Errors    []RPCErrorPortConfig `xml:"rpc-error,omitempty"`
		}{
			MessageID: msgID,
			Errors:    []RPCErrorPortConfig{{ErrorType: "application", ErrorTag: "operation-failed", ErrorSeverity: "error", ErrorMessage: "Failed to parse interface data from device"}},
		}
		xmlBytes, _ := xml.MarshalIndent(errorReply, "", "  ")
		return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
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
		if details.IfSpeed != nil {
			speedStr := strconv.Itoa(*details.IfSpeed)
			entry.Speed = &speedStr
		}

		// Description
		if details.IfDescription != "" {
			entry.Description = &details.IfDescription
		}

		// Switchport Info
		swConfig := SwitchportConfig{}
		changedSw := false
		if details.PortMode != nil && details.PortMode.Description != "" {
			mode := strings.ToLower(details.PortMode.Description)
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
					allowed := strings.Join(vlanStrings, ",")
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
		portConfigEntries = append(portConfigEntries, entry)
	}

	portConfigurations := PortConfigurations{
		Xmlns: "yang:port_config", // Set the desired short namespace for the GET response
		Ports: portConfigEntries,
	}

	// Use the new RpcReplyPortConfigGetData for the simplified GET response
	reply := RpcReplyPortConfigGetData{
		// XMLName is defined in the struct tag
		// MessageID is not included in this simplified response
		Data: &DataPortConfig{
			PortConfigurations: &portConfigurations,
		},
	}
	// Use marshalToXMLPortConfig, which now handles simplified rpc-reply
	return marshalToXMLPortConfig(reply, frameEnd)
}

// HandleLagGetConfig handles <get> for port-channel configurations
func HandleLagGetConfig(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{"uid": "Agent.Switch.Get.LAG.Table", "arg": nil},
		ID:     13, // Unique ID
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil || (miyagiResp != nil && miyagiResp.Error != nil) {
		errMsg := "Error communicating with device agent for LAG table"
		if err != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, err)
		} else if miyagiResp.Error != nil {
			errMsg = fmt.Sprintf("Device error for LAG table: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		}
		log.Printf("NETCONF_PORT_CFG_HANDLER (LAG): %s", errMsg)
		return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	var miyagiLagTable map[string]MiyagiLagDetail
	if err := json.Unmarshal(miyagiResp.Result, &miyagiLagTable); err != nil {
		log.Printf("NETCONF_PORT_CFG_HANDLER (LAG): Error unmarshalling LAG table: %v. Raw: %s", err, string(miyagiResp.Result))
		return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", "Failed to parse LAG data from device", frameEnd)
	}

	var lagEntries []LagEntryGetResponse
	for name, details := range miyagiLagTable {
		entry := LagEntryGetResponse{
			Name:     name,
			LacpMode: details.LacpMode,
		}
		if len(details.Members) > 0 {
			entry.Members = &Members{Member: details.Members}
		}
		lagEntries = append(lagEntries, entry)
	}

	// Sort for consistent output
	sort.Slice(lagEntries, func(i, j int) bool {
		return lagEntries[i].Name < lagEntries[j].Name
	})

	responseRoot := LagGetResponseRoot{PortChannels: lagEntries}
	xmlBytes, err := xml.MarshalIndent(responseRoot, "", "  ")
	if err != nil {
		log.Printf("NETCONF_PORT_CFG_HANDLER (LAG): Error marshalling LAG XML: %v", err)
		return buildErrorResponseBytesPortConfig(msgID, "application", "internal-error", "Error generating LAG XML response", frameEnd)
	}
	// For custom XML, we directly return the marshalled bytes with header and frameEnd
	return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
}

// HandleLagEditConfig handles <edit-config> for port-channel configurations
func HandleLagEditConfig(miyagiSocketPath string, request []byte, msgID, frameEnd string) []byte {
	var editReq EditConfigLagPayload
	configStartIndex := bytes.Index(request, []byte("<config>"))
	configEndIndex := bytes.LastIndex(request, []byte("</config>"))
	if configStartIndex == -1 || configEndIndex == -1 || configStartIndex >= configEndIndex {
		return buildErrorResponseBytesPortConfig(msgID, "protocol", "malformed-message", "Malformed <edit-config> for LAG", frameEnd)
	}
	configPayload := request[configStartIndex : configEndIndex+len("</config>")]

	if err := xml.Unmarshal(configPayload, &editReq); err != nil {
		log.Printf("NETCONF_PORT_CFG_HANDLER (LAG): Error unmarshalling <edit-config> payload: %v. Payload: %s", err, string(configPayload))
		return buildErrorResponseBytesPortConfig(msgID, "protocol", "malformed-message", "Invalid LAG configuration format", frameEnd)
	}

	for _, pc := range editReq.LagConfigurations.PortChannels {
		switch pc.Operation {
		case "create":
			err := callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.LAG.Create", map[string]interface{}{"name": pc.Name}, msgID, pc.Name, "LAG create")
			if err != nil {
				return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
			}
			if pc.LacpMode != nil {
				err = callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.LAG.LACP.Mode", map[string]interface{}{"lag_name": pc.Name, "mode": *pc.LacpMode}, msgID, pc.Name, "LACP mode")
				if err != nil {
					return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
				}
			}
			if pc.Members != nil {
				for _, member := range pc.Members.Member {
					if member.Operation == "add" || member.Operation == "" { // Default to add
						err = callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.LAG.AddMember", map[string]interface{}{"lag_name": pc.Name, "interface_name": member.Value}, msgID, pc.Name, "add member "+member.Value)
						if err != nil {
							return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
						}
					}
				}
			}
		case "delete":
			err := callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.LAG.Delete", map[string]interface{}{"name": pc.Name}, msgID, pc.Name, "LAG delete")
			if err != nil {
				return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
			}
		case "modify": // For modifying members or LACP mode of an existing LAG
			if pc.LacpMode != nil {
				err := callMiyagiPortConfig(miyagiSocketPath, "Agent.Switch.Set.LAG.LACP.Mode", map[string]interface{}{"lag_name": pc.Name, "mode": *pc.LacpMode}, msgID, pc.Name, "LACP mode")
				if err != nil {
					return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
				}
			}
			if pc.Members != nil {
				for _, member := range pc.Members.Member {
					uid := "Agent.Switch.Set.LAG.AddMember"
					if member.Operation == "remove" {
						uid = "Agent.Switch.Set.LAG.RemoveMember"
					}
					err := callMiyagiPortConfig(miyagiSocketPath, uid, map[string]interface{}{"lag_name": pc.Name, "interface_name": member.Value}, msgID, pc.Name, member.Operation+" member "+member.Value)
					if err != nil {
						return buildErrorResponseBytesPortConfig(msgID, "application", "operation-failed", err.Error(), frameEnd)
					}
				}
			}
		default:
			return buildErrorResponseBytesPortConfig(msgID, "protocol", "bad-attribute", fmt.Sprintf("Unsupported LAG operation: %s", pc.Operation), frameEnd)
		}
	}

	reply := RpcReplyPortConfig{Ok: &OkPortConfig{}} // Uses the simplified RpcReplyPortConfig
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
	// For custom GET responses that are already fully formed XML strings, this function might not be used.
	// This is primarily for RpcReplyPortConfig with Ok or Error.
	if err != nil {
		log.Printf("NETCONF_PORT_CFG_HANDLER: FATAL: Failed to marshal XML for simplified reply: %v", err)
		// Fallback for simplified rpc-reply
		return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rpc-reply><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`, frameEnd))
	}
	// If data is RpcReplyPortConfig (simplified), it won't have xmlns or message-id by default.
	return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
}

func buildErrorResponseBytesPortConfig(msgID, errType, errTag, errMsg, frameEnd string) []byte {
	escapedErrMsg := strings.ReplaceAll(errMsg, "<", "&lt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, ">", "&gt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, "&", "&amp;")
	reply := RpcReplyPortConfig{ // Uses the simplified RpcReplyPortConfig
		// MessageID removed
		Errors: []RPCErrorPortConfig{{ErrorType: errType, ErrorTag: errTag, ErrorSeverity: "error", ErrorMessage: escapedErrMsg}},
	}
	return marshalToXMLPortConfig(reply, frameEnd)
}

// Note: MiyagiInterfaceDetail struct is defined in interface.go.
// It's assumed to be accessible here as they are in the same 'handlers' package.
