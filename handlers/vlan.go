package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"qn-netconf/miyagi" // Assuming your miyagi client is in "net_conf/miyagi"
)

// VlanNamespace is the XML namespace for VLAN configuration.
const VlanNamespace = "urn:example:params:xml:ns:yang:vlan"

// InitVlanDB is called during server startup.
// For a Miyagi-backed system, this might just log or perform other setup.
func InitVlanDB() {
	log.Println("NETCONF_VLAN_HANDLER: Initialized to use Miyagi backend for VLANs.")
}

// BuildGetVlansResponse constructs the NETCONF rpc-reply for a get-vlans request.
func BuildGetVlansResponse(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.VLAN.Table",
			"arg": nil,
		},
		ID: 1, // Consider using unique IDs for requests
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_VLAN_HANDLER: Error calling Miyagi for Get.VLAN.Table: %v", err)
		return buildErrorResponse(msgID, "operation-failed", fmt.Sprintf("Failed to retrieve VLANs from device: %v", err), frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error retrieving VLANs: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_VLAN_HANDLER: Miyagi returned error for Get.VLAN.Table: %s", errMsg)
		return buildErrorResponse(msgID, "operation-failed", errMsg, frameEnd)
	}

	// Miyagi's "Agent.Switch.Get.VLAN.Table" returns a direct map of VLAN ID (string) to VLAN Name (string)
	// within its "result" field. Example miyagiResp.Result (json.RawMessage): `{"1":"Default","2":"Account",...}`
	var vlanMapResult map[string]string
	if err := json.Unmarshal(miyagiResp.Result, &vlanMapResult); err != nil {
		log.Printf("NETCONF_VLAN_HANDLER: Error unmarshalling Miyagi response for Get.VLAN.Table: %v. Raw: %s", err, string(miyagiResp.Result))
		return buildErrorResponse(msgID, "operation-failed", "Failed to parse VLAN data from device", frameEnd)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<vlans xmlns=\"%s\">", VlanNamespace))
	if len(vlanMapResult) == 0 {
		log.Println("NETCONF_VLAN_HANDLER: Miyagi returned no VLANs for Get.VLAN.Table.")
	}
	for vlanID, vlanName := range vlanMapResult {
		// NETCONF models often have a 'status'. Miyagi's Get.VLAN.Table doesn't provide it.
		// We'll use a default "active" status. This could be made configurable or omitted if the YANG model allows.
		sb.WriteString(fmt.Sprintf(
			"<vlan><id>%s</id><name>%s</name><status>active</status></vlan>",
			vlanID, vlanName,
		))
	}
	sb.WriteString("</vlans>")

	return []byte(fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  %s
</rpc-reply>
%s`, msgID, sb.String(), frameEnd,
	))
}

// HandleEditConfig processes an edit-config request for VLANs.
func HandleEditConfig(miyagiSocketPath string, request []byte, msgID, frameEnd string) []byte {
	parsedVlans, ok := parseVlanConfig(request) // This parses the NETCONF XML
	if !ok {
		return buildErrorResponse(msgID, "invalid-value", "Malformed VLAN configuration in edit-config", frameEnd)
	}

	// Simplified: treats entries as "create or update". A full implementation
	// would check nc:operation attributes (merge, create, replace, delete).
	for vlanIDStr, configData := range parsedVlans {
		vlanName := configData["name"]
		// status := configData["status"] // Status from XML might not be directly settable via Miyagi create.

		vlanIDInt, err := strconv.Atoi(vlanIDStr)
		if err != nil {
			log.Printf("NETCONF_VLAN_HANDLER: Invalid VLAN ID '%s' in edit-config", vlanIDStr)
			return buildErrorResponse(msgID, "invalid-value", fmt.Sprintf("Invalid VLAN ID '%s'", vlanIDStr), frameEnd)
		}

		miyagiReq := miyagi.MiyagiRequest{
			Method: "call",
			Params: map[string]interface{}{
				"uid": "Agent.Switch.Set.VLAN.Create",
				"arg": map[string]interface{}{
					"name":    vlanName,
					"vlan_id": vlanIDInt,
				},
			},
			ID: 1, // Consider unique IDs
		}

		miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to configure VLAN %d (%s) on device: %v", vlanIDInt, vlanName, err)
			log.Printf("NETCONF_VLAN_HANDLER: Error calling Miyagi for Set.VLAN.Create (VLAN %s): %v", vlanIDStr, err)
			return buildErrorResponse(msgID, "operation-failed", errMsg, frameEnd)
		}

		if miyagiResp.Error != nil {
			errMsg := fmt.Sprintf("Device error configuring VLAN %d (%s): %s (code: %d)", vlanIDInt, vlanName, miyagiResp.Error.Message, miyagiResp.Error.Code)
			log.Printf("NETCONF_VLAN_HANDLER: Miyagi returned error for Set.VLAN.Create (VLAN %s): %s", vlanIDStr, errMsg)
			return buildErrorResponse(msgID, "operation-failed", errMsg, frameEnd)
		}
		log.Printf("NETCONF_VLAN_HANDLER: Successfully processed edit-config for VLAN ID %s, Name: %s", vlanIDStr, vlanName)
	}

	return buildOKResponse(msgID, frameEnd)
}

// parseVlanConfig extracts VLAN configurations from the NETCONF XML <edit-config> payload.
// It expects the VLANs to be within <config><vlans xmlns="..."><vlan>...</vlan></vlans></config>.
func parseVlanConfig(request []byte) (map[string]map[string]string, bool) {
	configs := make(map[string]map[string]string)
	configTag := []byte("<config>")
	vlansTagStart := []byte(fmt.Sprintf("<vlans xmlns=\"%s\">", VlanNamespace))
	vlansTagEnd := []byte("</vlans>")
	vlanEntryStart := []byte("<vlan>")
	vlanEntryEnd := []byte("</vlan>")

	configIdx := bytes.Index(request, configTag)
	if configIdx == -1 {
		return nil, false // No <config> tag
	}
	configContent := request[configIdx+len(configTag):]

	vlansIdx := bytes.Index(configContent, vlansTagStart)
	if vlansIdx == -1 {
		return nil, false // No <vlans> tag with correct namespace
	}
	data := configContent[vlansIdx+len(vlansTagStart):]
	endVlansIdx := bytes.Index(data, vlansTagEnd)
	if endVlansIdx == -1 {
		return nil, false // Malformed <vlans>
	}
	data = data[:endVlansIdx]

	for {
		idx := bytes.Index(data, vlanEntryStart)
		if idx == -1 {
			break
		}
		data = data[idx+len(vlanEntryStart):]

		endIdx := bytes.Index(data, vlanEntryEnd)
		if endIdx == -1 {
			return nil, false // Malformed <vlan> entry
		}

		vlanData := data[:endIdx]
		data = data[endIdx+len(vlanEntryEnd):]

		id, name, status := parseVlanEntry(vlanData)
		if id == "" { // ID is mandatory
			return nil, false
		}
		configs[id] = map[string]string{"name": name, "status": status}
	}
	return configs, true
}

// parseVlanEntry extracts id, name, and status from a <vlan>...</vlan> XML snippet.
func parseVlanEntry(vlanData []byte) (id, name, status string) {
	extract := func(data []byte, tag string) string {
		openTag := []byte("<" + tag + ">")
		closeTag := []byte("</" + tag + ">")
		startIdx := bytes.Index(data, openTag)
		if startIdx == -1 {
			return ""
		}
		contentStart := startIdx + len(openTag)
		endIdx := bytes.Index(data[contentStart:], closeTag)
		if endIdx == -1 {
			return ""
		}
		return string(data[contentStart : contentStart+endIdx])
	}

	id = extract(vlanData, "id")
	name = extract(vlanData, "name")
	status = extract(vlanData, "status")
	return
}

func buildOKResponse(msgID, frameEnd string) []byte {
	return []byte(fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <ok/>
</rpc-reply>
%s`, msgID, frameEnd,
	))
}

func buildErrorResponse(msgID, errTag, errMsg, frameEnd string) []byte {
	// Ensure error message is XML-safe (basic escaping)
	escapedErrMsg := strings.ReplaceAll(errMsg, "<", "&lt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, ">", "&gt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, "&", "&amp;")

	return []byte(fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <rpc-error>
    <error-type>application</error-type>
    <error-tag>%s</error-tag>
    <error-severity>error</error-severity>
    <error-message xml:lang="en">%s</error-message>
  </rpc-error>
</rpc-reply>
%s`, msgID, errTag, escapedErrMsg, frameEnd,
	))
}
