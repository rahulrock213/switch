package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml" // Import encoding/xml
	"fmt"
	"log"
	"sort"
	"strconv"

	// "strings" // May not need strings as much

	"qn-netconf/miyagi" // Assuming your miyagi client is in "net_conf/miyagi"
)

// VlanNamespace is the XML namespace for VLAN configuration.
const VlanNamespace = "urn:example:params:xml:ns:yang:vlan"
const NetconfBaseNamespace = "urn:ietf:params:xml:ns:netconf:base:1.0"

// --- XML Data Structures ---

// RpcReply is the generic NETCONF rpc-reply structure.
type RpcReply struct {
	XMLName   xml.Name   `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
	MessageID string     `xml:"message-id,attr"`
	Data      *Data      `xml:"data,omitempty"`
	Ok        *Ok        `xml:"ok,omitempty"`
	Errors    []RPCError `xml:"rpc-error,omitempty"`
}

// Ok represents the <ok/> element.
type Ok struct {
	XMLName xml.Name `xml:"ok"`
}

// Data wraps the actual data payload.
type Data struct {
	XMLName   xml.Name    `xml:"data"`
	VlansData interface{} `xml:",innerxml"` // Use innerxml to embed pre-formatted or dynamic XML
}

// VlansHolder is the container for a list of VLANs in responses.
type VlansHolder struct {
	XMLName xml.Name    `xml:"urn:example:params:xml:ns:yang:vlan vlans"`
	Vlan    []VlanEntry `xml:"vlan"`
}

// VlanEntry represents a single VLAN's data for responses.
type VlanEntry struct {
	XMLName xml.Name `xml:"vlan"` // No namespace here, parent <vlans> has it
	ID      int      `xml:"id"`
	Name    string   `xml:"name"`
}

// RPCError represents a NETCONF rpc-error.
type RPCError struct {
	XMLName       xml.Name `xml:"rpc-error"`
	ErrorType     string   `xml:"error-type"`     // e.g., "application", "protocol"
	ErrorTag      string   `xml:"error-tag"`      // e.g., "invalid-value", "operation-failed"
	ErrorSeverity string   `xml:"error-severity"` // e.g., "error"
	ErrorMessage  string   `xml:"error-message"`  // Human-readable message
}

// --- For parsing <edit-config> ---

// EditConfigPayload represents the top-level structure of an <edit-config> request's <config> part.
type EditConfigPayload struct {
	XMLName xml.Name    `xml:"config"`
	Vlans   VlansConfig `xml:"urn:example:params:xml:ns:yang:vlan vlans"`
}

// VlansConfig is the <vlans> container within an <edit-config>.
type VlansConfig struct {
	XMLName     xml.Name          `xml:"urn:example:params:xml:ns:yang:vlan vlans"`
	VlanEntries []VlanConfigEntry `xml:"vlan"`
}

// VlanConfigEntry represents a <vlan> entry within an <edit-config>.
type VlanConfigEntry struct {
	XMLName   xml.Name `xml:"vlan"` // No namespace here, parent <vlans> has it
	ID        int      `xml:"id"`
	Name      string   `xml:"name"`
	Status    string   `xml:"status,omitempty"`
	Operation string   `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 operation,attr,omitempty"`
}

// InitVlanDB is called during server startup.
// For a Miyagi-backed system, this might just log or perform other setup.
func InitVlanDB() {
	log.Println("NETCONF_VLAN_HANDLER: Initialized to use Miyagi backend for VLANs.")
}

// marshalToXML is a helper to marshal structs to XML bytes with a standard prolog.
func marshalToXML(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_VLAN_HANDLER: FATAL: Failed to marshal XML: %v", err)
		// This is a programming error, should not happen with valid structs
		return []byte(fmt.Sprintf(
			`<?xml version="1.0" encoding="UTF-8"?><rpc-reply xmlns="%s"><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`,
			NetconfBaseNamespace, frameEnd,
		))
	}
	// Prepend XML declaration
	return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
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
		return buildErrorResponseBytes(msgID, "operation-failed", fmt.Sprintf("Failed to retrieve VLANs from device: %v", err), frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error retrieving VLANs: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_VLAN_HANDLER: Miyagi returned error for Get.VLAN.Table: %s", errMsg)
		return buildErrorResponseBytes(msgID, "operation-failed", errMsg, frameEnd)
	}

	// Miyagi's "Agent.Switch.Get.VLAN.Table" returns a direct map of VLAN ID (string) to VLAN Name (string)
	// within its "result" field. Example miyagiResp.Result (json.RawMessage): `{"1":"Default","2":"Account",...}`
	var vlanMapResult map[string]string
	if err := json.Unmarshal(miyagiResp.Result, &vlanMapResult); err != nil {
		log.Printf("NETCONF_VLAN_HANDLER: Error unmarshalling Miyagi response for Get.VLAN.Table: %v. Raw: %s", err, string(miyagiResp.Result))
		return buildErrorResponseBytes(msgID, "operation-failed", "Failed to parse VLAN data from device", frameEnd)
	}

	if len(vlanMapResult) == 0 {
		log.Println("NETCONF_VLAN_HANDLER: Miyagi returned no VLANs for Get.VLAN.Table.")
	}

	var vlanEntries []VlanEntry
	for vlanIDStr, vlanName := range vlanMapResult {
		vlanIDInt, err := strconv.Atoi(vlanIDStr)
		if err != nil {
			// This case should ideally not happen if Miyagi returns valid integer strings as keys
			log.Printf("NETCONF_VLAN_HANDLER: Invalid VLAN ID format '%s' from Miyagi: %v. Skipping.", vlanIDStr, err)
			continue
		}
		vlanEntries = append(vlanEntries, VlanEntry{ID: vlanIDInt, Name: vlanName})
	}

	// Sort the list by VLAN ID (integer value)
	sort.Slice(vlanEntries, func(i, j int) bool {
		return vlanEntries[i].ID < vlanEntries[j].ID
	})

	vlansData := VlansHolder{Vlan: vlanEntries}
	// Marshal VlansHolder to XML bytes first
	vlansXMLBytes, err := xml.MarshalIndent(vlansData, "    ", "  ") // Indent within <data>
	if err != nil {
		log.Printf("NETCONF_VLAN_HANDLER: Error marshalling VlansHolder: %v", err)
		return buildErrorResponseBytes(msgID, "operation-failed", "Failed to format VLAN data for XML response", frameEnd)
	}

	reply := RpcReply{
		MessageID: msgID,
		Data: &Data{
			VlansData: vlansXMLBytes, // Embed the marshalled <vlans>...</vlans>
		},
	}
	return marshalToXML(reply, frameEnd)
}

// HandleEditConfig processes an edit-config request for VLANs.
func HandleEditConfig(miyagiSocketPath string, request []byte, msgID, frameEnd string) []byte {
	// Log the first 200 bytes of the request to see what HandleEditConfig is receiving
	requestSnippet := string(request)
	if len(requestSnippet) > 200 {
		requestSnippet = requestSnippet[:200] + "..."
	}
	log.Printf("NETCONF_VLAN_HANDLER: HandleEditConfig called for msgID %s. Request snippet: %s", msgID, requestSnippet)

	// The request is the full NETCONF message, e.g. <rpc><edit-config>...</edit-config></rpc>
	// We need to extract the <config> part to unmarshal into EditConfigPayload.
	// The original parseVlanConfig looked for <config><vlans xmlns="...">

	configTagStart := []byte("<config>")
	configTagEnd := []byte("</config>")

	configStartIdx := bytes.Index(request, configTagStart)
	if configStartIdx == -1 {
		log.Printf("NETCONF_VLAN_HANDLER: <config> tag not found in request for msgID %s.", msgID)
		return buildErrorResponseBytes(msgID, "malformed-message", "Missing <config> element in edit-config", frameEnd)
	}

	configEndIdx := bytes.Index(request[configStartIdx:], configTagEnd)
	if configEndIdx == -1 {
		log.Printf("NETCONF_VLAN_HANDLER: Closing </config> tag not found for msgID %s.", msgID)
		return buildErrorResponseBytes(msgID, "malformed-message", "Malformed <config> element in edit-config", frameEnd)
	}

	configContent := request[configStartIdx : configStartIdx+configEndIdx+len(configTagEnd)]

	var payload EditConfigPayload
	err := xml.Unmarshal(configContent, &payload)
	if err != nil {
		log.Printf("NETCONF_VLAN_HANDLER: Failed to unmarshal <config> XML for msgID %s: %v. Content: %s", msgID, err, string(configContent))
		return buildErrorResponseBytes(msgID, "invalid-value", fmt.Sprintf("Malformed VLAN configuration in edit-config: %v", err), frameEnd)
	}

	if payload.Vlans.XMLName.Local == "" { // Check if Vlans was parsed
		log.Printf("NETCONF_VLAN_HANDLER: <vlans> section not found or not parsed in <config> for msgID %s.", msgID)
		return buildErrorResponseBytes(msgID, "invalid-value", "Missing or malformed <vlans> section in configuration", frameEnd)
	}

	log.Printf("NETCONF_VLAN_HANDLER: XML unmarshal successful for msgID %s. Parsed %d VLAN entries.", msgID, len(payload.Vlans.VlanEntries))

	if len(payload.Vlans.VlanEntries) == 0 {
		log.Printf("NETCONF_VLAN_HANDLER: No VLAN entries found in the parsed config for msgID %s.", msgID)
	}

	// Simplified: treats entries as "create or update". A full implementation
	// would check nc:operation attributes (merge, create, replace, delete).
	for _, vlanEntry := range payload.Vlans.VlanEntries {
		log.Printf("NETCONF_VLAN_HANDLER: Processing edit-config for VLAN ID %d, Name: %s, Operation: %s", vlanEntry.ID, vlanEntry.Name, vlanEntry.Operation)

		if vlanEntry.ID == 0 { // Basic validation
			log.Printf("NETCONF_VLAN_HANDLER: Invalid VLAN ID '0' in edit-config for msgID %s", msgID)
			return buildErrorResponseBytes(msgID, "invalid-value", "VLAN ID '0' is not allowed", frameEnd)
		}

		miyagiReq := miyagi.MiyagiRequest{
			Method: "call",
			Params: map[string]interface{}{
				"uid": "Agent.Switch.Set.VLAN.Create",
				"arg": map[string]interface{}{
					"name":    vlanEntry.Name,
					"vlan_id": vlanEntry.ID,
				},
			},
			ID: 1, // Consider unique IDs
		}

		miyagiResp, miyagiErr := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
		if miyagiErr != nil {
			errMsg := fmt.Sprintf("Failed to configure VLAN %d (%s) on device: %v", vlanEntry.ID, vlanEntry.Name, miyagiErr)
			log.Printf("NETCONF_VLAN_HANDLER: Error calling Miyagi for Set.VLAN.Create (VLAN %d): %v", vlanEntry.ID, miyagiErr)
			return buildErrorResponseBytes(msgID, "operation-failed", errMsg, frameEnd)
		}

		if miyagiResp.Error != nil {
			errMsg := fmt.Sprintf("Device error configuring VLAN %d (%s): %s (code: %d)", vlanEntry.ID, vlanEntry.Name, miyagiResp.Error.Message, miyagiResp.Error.Code)
			log.Printf("NETCONF_VLAN_HANDLER: Miyagi returned error for Set.VLAN.Create (VLAN %d): %s", vlanEntry.ID, errMsg)
			return buildErrorResponseBytes(msgID, "operation-failed", errMsg, frameEnd)
		}
		log.Printf("NETCONF_VLAN_HANDLER: Successfully processed edit-config for VLAN ID %d, Name: %s", vlanEntry.ID, vlanEntry.Name)
	}

	return buildOKResponseBytes(msgID, frameEnd)
}

// buildOKResponseBytes creates a NETCONF <ok/> response.
func buildOKResponseBytes(msgID, frameEnd string) []byte {
	reply := RpcReply{
		MessageID: msgID,
		Ok:        &Ok{},
	}
	return marshalToXML(reply, frameEnd)
}

// buildErrorResponseBytes creates a NETCONF <rpc-error> response.
func buildErrorResponseBytes(msgID, errTag, errMsg, frameEnd string) []byte {
	// XML marshaler handles escaping of characters like <, >, & in string fields.
	reply := RpcReply{
		MessageID: msgID,
		Errors: []RPCError{
			{
				ErrorType:     "application", // Or "protocol", "rpc", "transport"
				ErrorTag:      errTag,
				ErrorSeverity: "error",
				ErrorMessage:  errMsg,
			},
		},
	}
	return marshalToXML(reply, frameEnd)
}
