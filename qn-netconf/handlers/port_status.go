package handlers

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html" // For escaping XML in error messages
	"log"
	"sync/atomic" // For a simple ID generator

	"qn-netconf/miyagi"
)

const PortStatusNamespace = "yang:get_port_status"
const PortStatusCapability = "yang:get_port_status" // Same as namespace for capability advertisement

// --- NETCONF Request Parsing Structures ---

// GetPortStatusRequestFilter is used to unmarshal the content of <port-status> from the filter.
type GetPortStatusRequestFilter struct {
	XMLName         xml.Name `xml:"port-status"` // Expects <port-status>
	Xmlns           string   `xml:"xmlns,attr,omitempty"`
	InterfaceNumber int      `xml:"interface-number"`
}

// GetFilterPayload is used to unmarshal the <filter> element.
type GetFilterPayload struct {
	XMLName xml.Name                   `xml:"filter"`
	Content GetPortStatusRequestFilter `xml:"port-status"` // Expects <port-status> inside <filter>
}

// GetPayload is used to unmarshal the <get> element.
type GetPayload struct {
	XMLName    xml.Name                    `xml:"get"`
	Filter     *GetFilterPayload           `xml:"filter,omitempty"`      // For <get><filter><port-status>...</port-status></filter></get>
	PortStatus *GetPortStatusRequestFilter `xml:"port-status,omitempty"` // For <get><port-status>...</port-status></get>
}

// FullRpcRequestForPortStatus is used to unmarshal the entire <rpc> request.
type FullRpcRequestForPortStatus struct {
	XMLName xml.Name   `xml:"rpc"`
	Get     GetPayload `xml:"get"`
}

// --- NETCONF Response XML Structures ---

// OpStatus represents the <status><value>...</value><description>...</description></status> block.
type OpStatus struct {
	Value       int    `xml:"value"`
	Description string `xml:"description"`
}

// PortStatusPayload is the main data structure for the <port-status> element in the response.
type PortStatusPayload struct {
	XMLName         xml.Name `xml:"port-status"`      // Tag name for the XML element
	Xmlns           string   `xml:"xmlns,attr"`       // Namespace attribute for <port-status>
	InterfaceNumber int      `xml:"interface-number"` // <interface-number>
	Status          OpStatus `xml:"status"`           // <status>...</status>
}

// RpcReplyPortStatusGet is the top-level <rpc-reply> for the get port status operation.
type RpcReplyPortStatusGet struct {
	XMLName   xml.Name  `xml:"rpc-reply"`
	MessageID string    `xml:"-"` //  Omit the MessageID field
	Data      *struct { // Anonymous struct for the <data> element
		XMLName           xml.Name `xml:"data"`
		PortStatusPayload PortStatusPayload
	} `xml:"data,omitempty"`
	Errors []RPCError `xml:"rpc-error,omitempty"`
}

// --- Miyagi JSON RPC Structs ---
// These are simplified as MiyagiRequest and MiyagiResponse are handled by the miyagi package.

type InterfaceStatusArg struct {
	InterfaceNumber int `json:"interface_number"`
}

var miyagiRequestCounter uint32 // For local ID generation

// HandleGetPortStatus processes a NETCONF <get> request for port status.
func HandleGetPortStatus(miyagiSocketPath string, requestXML []byte, msgID string, frameEnd string) []byte {
	var parsedReq FullRpcRequestForPortStatus
	if err := xml.Unmarshal(requestXML, &parsedReq); err != nil {
		log.Printf("NETCONF_PORT_STATUS_HANDLER: Error unmarshalling request: %v. Request: %s", err, string(requestXML))
		return buildErrorResponseBytesCommon(msgID, "malformed-message", "Invalid request format", frameEnd)
	}

	var portStatusFilterCriteria *GetPortStatusRequestFilter

	if parsedReq.Get.PortStatus != nil && parsedReq.Get.PortStatus.XMLName.Local == "port-status" {
		// Case 1: <get><port-status>...</port-status></get>
		portStatusFilterCriteria = parsedReq.Get.PortStatus
		log.Printf("NETCONF_PORT_STATUS_HANDLER: Found <port-status> directly under <get>.")
	} else if parsedReq.Get.Filter != nil && parsedReq.Get.Filter.Content.XMLName.Local == "port-status" {
		// Case 2: <get><filter><port-status>...</port-status></filter></get>
		portStatusFilterCriteria = &parsedReq.Get.Filter.Content
		log.Printf("NETCONF_PORT_STATUS_HANDLER: Found <port-status> under <get><filter>.")
	} else {
		log.Printf("NETCONF_PORT_STATUS_HANDLER: Malformed request, <port-status> filter not found. Request: %s", string(requestXML))
		return buildErrorResponseBytesCommon(msgID, "malformed-message", "Missing or malformed <port-status> filter criteria", frameEnd)
	}

	// Check if the namespace in the filter matches what we expect.
	// main.go's dispatch logic already performs a basic namespace check.
	if portStatusFilterCriteria.Xmlns != PortStatusNamespace {
		log.Printf("NETCONF_PORT_STATUS_HANDLER: Namespace mismatch in <port-status> filter. Expected '%s', got '%s'", PortStatusNamespace, portStatusFilterCriteria.Xmlns)
		// Depending on strictness, you might return an error here.
		// For now, proceeding as main.go should have caught critical mismatches.
	}

	interfaceNum := portStatusFilterCriteria.InterfaceNumber
	if interfaceNum <= 0 {
		log.Printf("NETCONF_PORT_STATUS_HANDLER: Invalid interface number %d from filter.", interfaceNum)
		return buildErrorResponseBytesCommon(msgID, "invalid-value", "Interface number must be positive", frameEnd)
	}

	log.Printf("NETCONF_PORT_STATUS_HANDLER: Querying status for interface number: %d", interfaceNum)

	miyagiReqPayload := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.Interface.Status",
			"arg": InterfaceStatusArg{
				InterfaceNumber: interfaceNum,
			},
		},
		ID: int(generateMiyagiID()), // Use local ID generator
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReqPayload)
	if err != nil {
		log.Printf("NETCONF_PORT_STATUS_HANDLER: Miyagi call failed for Agent.Switch.Get.Interface.Status (interface %d): %v", interfaceNum, err)
		return buildErrorResponseBytesCommon(msgID, "operation-failed", "Error communicating with device agent", frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error for interface %d status: %s (code: %d)", interfaceNum, miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_PORT_STATUS_HANDLER: %s", errMsg)
		return buildErrorResponseBytesCommon(msgID, "operation-failed", errMsg, frameEnd)
	}

	if miyagiResp.Result == nil {
		log.Printf("NETCONF_PORT_STATUS_HANDLER: Miyagi returned no result for interface %d status.", interfaceNum)
		return buildErrorResponseBytesCommon(msgID, "operation-failed", "No result from device agent", frameEnd)
	}

	var miyagiResultInt int
	if err := json.Unmarshal(miyagiResp.Result, &miyagiResultInt); err != nil {
		log.Printf("NETCONF_PORT_STATUS_HANDLER: Error unmarshalling Miyagi result for interface %d: %v. Raw: %s", interfaceNum, err, string(miyagiResp.Result))
		return buildErrorResponseBytesCommon(msgID, "operation-failed", "Invalid data format from device agent", frameEnd)
	}

	var statusStr string
	statusVal := miyagiResultInt

	switch statusVal {
	case 1:
		statusStr = "UP"
	case 2:
		statusStr = "DOWN"
	case 3:
		statusStr = "TESTING"
	default:
		statusStr = "UNKNOWN"
	}

	responsePayload := PortStatusPayload{
		Xmlns:           PortStatusNamespace, // Set the namespace for the <port-status> tag
		InterfaceNumber: interfaceNum,
		Status:          OpStatus{Value: statusVal, Description: statusStr},
	}

	reply := RpcReplyPortStatusGet{
		MessageID: msgID,
		Data: &struct {
			XMLName           xml.Name `xml:"data"`
			PortStatusPayload PortStatusPayload
		}{PortStatusPayload: responsePayload},
	}

	return marshalToXMLCommon(reply, frameEnd)
}

// generateMiyagiID creates a new unique ID for Miyagi requests locally.
func generateMiyagiID() uint32 {
	return atomic.AddUint32(&miyagiRequestCounter, 1)
}

// marshalToXMLCommon is a helper to marshal structs to XML bytes with a standard prolog and frame end.
func marshalToXMLCommon(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_PORT_STATUS_HANDLER: FATAL: Failed to marshal XML: %v", err)
		// Fallback to a very basic error string if marshalling the error struct itself fails
		return []byte(fmt.Sprintf(
			`<?xml version="1.0" encoding="UTF-8"?><rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`,
			frameEnd,
		))
	}
	// Prepend XML declaration, add a newline before frameEnd
	return append([]byte(xml.Header), append(append(xmlBytes, '\n'), []byte(frameEnd)...)...)
}

// buildErrorResponseBytesCommon creates a NETCONF <rpc-error> response.
func buildErrorResponseBytesCommon(msgID, errTag, errMsg, frameEnd string) []byte {
	// Escape XML special characters in the error message
	escapedErrMsg := html.EscapeString(errMsg)

	reply := RpcReplyPortStatusGet{
		MessageID: msgID,
		Errors: []RPCError{
			{
				// XMLName will be <rpc-error> due to struct tag
				ErrorType:     "application", // Or "protocol", "rpc", "transport"
				ErrorTag:      errTag,
				ErrorSeverity: "error",
				ErrorMessage:  escapedErrMsg,
			},
		},
	}
	return marshalToXMLCommon(reply, frameEnd)
}
