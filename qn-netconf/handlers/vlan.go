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
// Changed to align with the namespace observed in the client's <get> filter.
const VlanNamespace = "yang:get_vlan"
const NetconfBaseNamespace = "yang:vlan"

// --- XML Data Structures ---

// RpcReply is the generic NETCONF rpc-reply structure.
type RpcReply struct {
	// XMLName specifies the namespace and local name for <rpc-reply>, aligning with NETCONF base.
	XMLName xml.Name     `xml:"yang:vlan rpc-reply"`
	Vlans   *VlansHolder `xml:"vlans,omitempty"` // VlansHolder will be directly under rpc-reply
	Result  string       `xml:"result,omitempty"`
	Errors  []RPCError   `xml:"rpc-error,omitempty"`
}

// VlansHolder is the container for a list of VLANs in responses.
type VlansHolder struct {
	// Aligned namespace with the VlanNamespace constant and client's request filter.
	XMLName xml.Name    `xml:"yang:get_vlan vlans"`
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
	XMLName xml.Name `xml:"config"`
	// Changed to be namespace-agnostic for the field matching.
	// This allows it to unmarshal <vlans> with either "yang:set_vlan"
	// or the original "urn:example:params:xml:ns:yang:vlan" namespace.
	Vlans VlansConfig `xml:"vlans"`
}

// VlansConfig is the <vlans> container within an <edit-config>.
type VlansConfig struct {
	// Changed XMLName to be namespace-agnostic.
	// The actual namespace will be on the <vlans> tag in the XML,
	// and the unmarshaller will associate it with this struct.
	XMLName     xml.Name          `xml:"vlans"`
	VlanEntries []VlanConfigEntry `xml:"vlan"`
}

// VlanConfigEntry represents a <vlan> entry within an <edit-config>.
type VlanConfigEntry struct {
	XMLName   xml.Name `xml:"vlan"` // No namespace here, parent <vlans> has it
	ID        int      `xml:"id"`
	Name      string   `xml:"name"`
	Status    string   `xml:"status,omitempty"`
	Operation string   `xml:"yang:vlan operation,attr,omitempty"`
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
	// Prepend XML declaration, add a newline before frameEnd
	return append([]byte(xml.Header), append(append(xmlBytes, '\n'), []byte(frameEnd)...)...)
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
		return buildErrorResponseBytes("operation-failed", fmt.Sprintf("Failed to retrieve VLANs from device: %v", err), frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error retrieving VLANs: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_VLAN_HANDLER: Miyagi returned error for Get.VLAN.Table: %s", errMsg)
		return buildErrorResponseBytes("operation-failed", errMsg, frameEnd)
	}

	// Miyagi's "Agent.Switch.Get.VLAN.Table" returns a direct map of VLAN ID (string) to VLAN Name (string)
	// within its "result" field. Example miyagiResp.Result (json.RawMessage): `{"1":"Default","2":"Account",...}`
	var vlanMapResult map[string]string
	if err := json.Unmarshal(miyagiResp.Result, &vlanMapResult); err != nil {
		log.Printf("NETCONF_VLAN_HANDLER: Error unmarshalling Miyagi response for Get.VLAN.Table (msgID: %s): %v. Raw: %s", msgID, err, string(miyagiResp.Result))
		return buildErrorResponseBytes("operation-failed", "Failed to parse VLAN data from device", frameEnd)
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

	reply := RpcReply{
		// Directly embed the VlansHolder. Its XMLName will be used.
		Vlans: &vlansData,
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

	// Modified to parse <vlans> directly from <edit-config>, bypassing the <config> wrapper.
	// Client should send <edit-config><vlans>...</vlans></edit-config>

	// A more robust way to find the start of <vlans> tag, accommodating attributes.
	vlansTagStartBytes := []byte("<vlans")
	vlansTagEndBytes := []byte("</vlans>")

	vlansStartIdx := bytes.Index(request, vlansTagStartBytes)
	if vlansStartIdx == -1 {
		log.Printf("NETCONF_VLAN_HANDLER: <vlans> tag not found in request for msgID %s.", msgID)
		return buildErrorResponseBytes("malformed-message", "Missing <vlans> element in edit-config", frameEnd)
	}

	// Find the closing tag for <vlans> starting from after the opening tag is found.
	vlansEndIdx := bytes.Index(request[vlansStartIdx:], vlansTagEndBytes)
	if vlansEndIdx == -1 {
		log.Printf("NETCONF_VLAN_HANDLER: Closing </vlans> tag not found for msgID %s.", msgID)
		return buildErrorResponseBytes("malformed-message", "Malformed <vlans> element in edit-config", frameEnd)
	}
	// Adjust vlansEndIdx to be relative to the original request byte slice
	vlansEndIdx += vlansStartIdx

	vlansContent := request[vlansStartIdx : vlansEndIdx+len(vlansTagEndBytes)]

	var vlansData VlansConfig // Unmarshal directly into VlansConfig
	err := xml.Unmarshal(vlansContent, &vlansData)
	if err != nil {
		log.Printf("NETCONF_VLAN_HANDLER: Failed to unmarshal <vlans> XML for msgID %s: %v. Content: %s", msgID, err, string(vlansContent))
		return buildErrorResponseBytes("invalid-value", fmt.Sprintf("Malformed VLAN configuration in edit-config: %v", err), frameEnd)
	}
	if vlansData.XMLName.Local == "" { // Check if VlansConfig was properly unmarshalled
		log.Printf("NETCONF_VLAN_HANDLER: <vlans> section not found or not parsed in request for msgID %s.", msgID)
		return buildErrorResponseBytes("invalid-value", "Missing or malformed <vlans> section in configuration", frameEnd)
	}

	log.Printf("NETCONF_VLAN_HANDLER: XML unmarshal successful for msgID %s. Parsed %d VLAN entries.", msgID, len(vlansData.VlanEntries))

	if len(vlansData.VlanEntries) == 0 {
		log.Printf("NETCONF_VLAN_HANDLER: No VLAN entries found in the parsed config for msgID %s.", msgID)
	}

	// Simplified: treats entries as "create or update". A full implementation
	// would check nc:operation attributes (merge, create, replace, delete).
	for _, vlanEntry := range vlansData.VlanEntries {
		log.Printf("NETCONF_VLAN_HANDLER: Processing edit-config for VLAN ID %d, Name: %s, Operation: %s", vlanEntry.ID, vlanEntry.Name, vlanEntry.Operation)

		if vlanEntry.ID == 0 { // Basic validation
			log.Printf("NETCONF_VLAN_HANDLER: Invalid VLAN ID '0' in edit-config for msgID %s", msgID)
			return buildErrorResponseBytes("invalid-value", "VLAN ID '0' is not allowed", frameEnd) //
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
			return buildErrorResponseBytes("operation-failed", errMsg, frameEnd)
		}

		if miyagiResp.Error != nil {
			errMsg := fmt.Sprintf("Device error configuring VLAN %d (%s): %s (code: %d)", vlanEntry.ID, vlanEntry.Name, miyagiResp.Error.Message, miyagiResp.Error.Code)
			log.Printf("NETCONF_VLAN_HANDLER: Miyagi returned error for Set.VLAN.Create (VLAN %d): %s", vlanEntry.ID, errMsg)
			return buildErrorResponseBytes("operation-failed", errMsg, frameEnd)
		}
		log.Printf("NETCONF_VLAN_HANDLER: Successfully processed edit-config for VLAN ID %d, Name: %s", vlanEntry.ID, vlanEntry.Name)
	}

	return buildOKResponseBytes(frameEnd)
}

// buildOKResponseBytes creates a NETCONF <ok/> response.
func buildOKResponseBytes(frameEnd string) []byte {
	reply := RpcReply{Result: "ok"} // Populate Result field instead of Ok
	return marshalToXML(reply, frameEnd)
}

// buildErrorResponseBytes creates a NETCONF <rpc-error> response.
func buildErrorResponseBytes(errTag, errMsg, frameEnd string) []byte {
	// XML marshaler handles escaping of characters like <, >, & in string fields.
	reply := RpcReply{
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
