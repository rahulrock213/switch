package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"qn-netconf/miyagi" // Assuming your miyagi client is in "net_conf/miyagi"
	"strings"
)

const InterfaceNamespace = "urn:example:params:xml:ns:yang:interfaces" // Example namespace

// BuildGetInterfacesResponse constructs the NETCONF rpc-reply for a get-interfaces request
func BuildGetInterfacesResponse(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.General.AllInterfaces",
			"arg": nil,
		},
		ID: 2, // Use a different ID or a generator
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_IF_HANDLER: Error calling Miyagi for Get.General.AllInterfaces: %v", err)
		return buildErrorResponse(msgID, "operation-failed", fmt.Sprintf("Failed to retrieve interfaces: %v", err), frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error retrieving interfaces: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_IF_HANDLER: Miyagi returned error for Get.General.AllInterfaces: %s", errMsg)
		return buildErrorResponse(msgID, "operation-failed", errMsg, frameEnd)
	}

	// The miyagiResp.Result for AllInterfaces is a map of interface names to their details.
	// We need to format this into XML. This is a simplified example.
	// A proper YANG model would define the XML structure.
	var interfaceData map[string]interface{} // Or a more specific struct
	if err := json.Unmarshal(miyagiResp.Result, &interfaceData); err != nil {
		log.Printf("NETCONF_IF_HANDLER: Error unmarshalling Miyagi interface data: %v. Raw: %s", err, string(miyagiResp.Result))
		return buildErrorResponse(msgID, "operation-failed", "Failed to parse interface data from device", frameEnd)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<interfaces xmlns=\"%s\">", InterfaceNamespace))
	for name, details := range interfaceData {
		// This is a very basic XML representation.
		// In a real scenario, you'd marshal the 'details' (which is likely a struct) into XML
		// according to your YANG model for interfaces.
		sb.WriteString(fmt.Sprintf("<interface><name>%s</name><details>%+v</details></interface>", name, details))
	}
	sb.WriteString("</interfaces>")

	return []byte(fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">%s</rpc-reply>%s`,
		msgID, sb.String(), frameEnd,
	))
}
