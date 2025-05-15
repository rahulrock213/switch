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

const PortSpeedNamespace = "yang:get_port_speed"
const PortSpeedCapability = "yang:get_port_speed" // Same as namespace for capability advertisement

// --- NETCONF Request Parsing Structures ---

// GetPortSpeedRequestFilter is used to unmarshal the content of <port-speed> from the filter.
type GetPortSpeedRequestFilter struct {
	XMLName         xml.Name `xml:"port-speed"` // Expects <port-speed>
	Xmlns           string   `xml:"xmlns,attr,omitempty"`
	InterfaceNumber int      `xml:"interface-number"`
}

// GetFilterPayloadSpeed is used to unmarshal the <filter> element for port speed.
type GetFilterPayloadSpeed struct {
	XMLName xml.Name                  `xml:"filter"`
	Content GetPortSpeedRequestFilter `xml:"port-speed"` // Expects <port-speed> inside <filter>
}

// GetPayloadSpeed is used to unmarshal the <get> element for port speed.
type GetPayloadSpeed struct {
	XMLName   xml.Name                   `xml:"get"`
	Filter    *GetFilterPayloadSpeed     `xml:"filter,omitempty"`     // For <get><filter><port-speed>...</port-speed></filter></get>
	PortSpeed *GetPortSpeedRequestFilter `xml:"port-speed,omitempty"` // For <get><port-speed>...</port-speed></get>
}

// FullRpcRequestForPortSpeed is used to unmarshal the entire <rpc> request.
type FullRpcRequestForPortSpeed struct {
	XMLName xml.Name        `xml:"rpc"`
	Get     GetPayloadSpeed `xml:"get"`
}

// --- NETCONF Response XML Structures ---

// PortSpeedPayload is the main data structure for the <port-speed> element in the response.
type PortSpeedPayload struct {
	XMLName         xml.Name `xml:"port-speed"`       // Tag name for the XML element
	Xmlns           string   `xml:"xmlns,attr"`       // Namespace attribute for <port-speed>
	InterfaceNumber int      `xml:"interface-number"` // <interface-number>
	Speed           int      `xml:"speed"`            // <speed>
}

// RpcReplyPortSpeedGet is the top-level <rpc-reply> for the get port speed operation.
type RpcReplyPortSpeedGet struct {
	XMLName   xml.Name  `xml:"rpc-reply"`
	MessageID string    `xml:"-"` // Omit the MessageID field
	Data      *struct { // Anonymous struct for the <data> element
		XMLName          xml.Name `xml:"data"`
		PortSpeedPayload PortSpeedPayload
	} `xml:"data,omitempty"`
	Errors []RPCError `xml:"rpc-error,omitempty"` // Assumes RPCError is defined in package scope (e.g. from vlan.go)
}

// --- Miyagi JSON RPC Structs ---
type InterfaceSpeedArg struct {
	InterfaceNumber int `json:"interface_number"`
}

var miyagiRequestCounterSpeed uint32 // For local ID generation

// HandleGetPortSpeed processes a NETCONF <get> request for port speed.
func HandleGetPortSpeed(miyagiSocketPath string, requestXML []byte, msgID string, frameEnd string) []byte {
	var parsedReq FullRpcRequestForPortSpeed
	if err := xml.Unmarshal(requestXML, &parsedReq); err != nil {
		log.Printf("NETCONF_PORT_SPEED_HANDLER: Error unmarshalling request: %v. Request: %s", err, string(requestXML))
		return buildErrorResponseBytesCommonSpeed(msgID, "malformed-message", "Invalid request format", frameEnd)
	}

	var portSpeedFilterCriteria *GetPortSpeedRequestFilter

	if parsedReq.Get.PortSpeed != nil && parsedReq.Get.PortSpeed.XMLName.Local == "port-speed" {
		portSpeedFilterCriteria = parsedReq.Get.PortSpeed
	} else if parsedReq.Get.Filter != nil && parsedReq.Get.Filter.Content.XMLName.Local == "port-speed" {
		portSpeedFilterCriteria = &parsedReq.Get.Filter.Content
	} else {
		log.Printf("NETCONF_PORT_SPEED_HANDLER: Malformed request, <port-speed> filter not found. Request: %s", string(requestXML))
		return buildErrorResponseBytesCommonSpeed(msgID, "malformed-message", "Missing or malformed <port-speed> filter criteria", frameEnd)
	}

	if portSpeedFilterCriteria.Xmlns != PortSpeedNamespace {
		log.Printf("NETCONF_PORT_SPEED_HANDLER: Namespace mismatch in <port-speed> filter. Expected '%s', got '%s'", PortSpeedNamespace, portSpeedFilterCriteria.Xmlns)
	}

	interfaceNum := portSpeedFilterCriteria.InterfaceNumber
	if interfaceNum <= 0 {
		log.Printf("NETCONF_PORT_SPEED_HANDLER: Invalid interface number %d from filter.", interfaceNum)
		return buildErrorResponseBytesCommonSpeed(msgID, "invalid-value", "Interface number must be positive", frameEnd)
	}

	log.Printf("NETCONF_PORT_SPEED_HANDLER: Querying speed for interface number: %d", interfaceNum)

	miyagiReqPayload := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.Interface.Speed",
			"arg": InterfaceSpeedArg{InterfaceNumber: interfaceNum},
		},
		ID: int(generateMiyagiIDSpeed()),
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReqPayload)
	if err != nil || miyagiResp.Error != nil {
		errMsg := "Error communicating with device agent for port speed"
		if err != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, err)
		} else {
			errMsg = fmt.Sprintf("Device error for port speed: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		}
		log.Printf("NETCONF_PORT_SPEED_HANDLER: %s", errMsg)
		return buildErrorResponseBytesCommonSpeed(msgID, "operation-failed", errMsg, frameEnd)
	}

	if miyagiResp.Result == nil {
		log.Printf("NETCONF_PORT_SPEED_HANDLER: Miyagi returned no result for interface %d speed.", interfaceNum)
		return buildErrorResponseBytesCommonSpeed(msgID, "operation-failed", "No result from device agent for port speed", frameEnd)
	}

	var speed int
	if err := json.Unmarshal(miyagiResp.Result, &speed); err != nil {
		log.Printf("NETCONF_PORT_SPEED_HANDLER: Error unmarshalling Miyagi result for interface %d speed: %v. Raw: %s", interfaceNum, err, string(miyagiResp.Result))
		return buildErrorResponseBytesCommonSpeed(msgID, "operation-failed", "Invalid data format from device agent for port speed", frameEnd)
	}

	responsePayload := PortSpeedPayload{
		Xmlns:           PortSpeedNamespace,
		InterfaceNumber: interfaceNum,
		Speed:           speed,
	}

	reply := RpcReplyPortSpeedGet{
		Data: &struct {
			XMLName          xml.Name `xml:"data"`
			PortSpeedPayload PortSpeedPayload
		}{PortSpeedPayload: responsePayload},
	}

	return marshalToXMLCommonSpeed(reply, frameEnd)
}

func generateMiyagiIDSpeed() uint32 {
	return atomic.AddUint32(&miyagiRequestCounterSpeed, 1)
}

func marshalToXMLCommonSpeed(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_PORT_SPEED_HANDLER: FATAL: Failed to marshal XML: %v", err)
		return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rpc-reply><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`, frameEnd))
	}
	return append([]byte(xml.Header), append(append(xmlBytes, '\n'), []byte(frameEnd)...)...)
}

func buildErrorResponseBytesCommonSpeed(msgID, errTag, errMsg, frameEnd string) []byte {
	escapedErrMsg := html.EscapeString(errMsg)
	reply := RpcReplyPortSpeedGet{ // Use the specific reply struct for port speed
		Errors: []RPCError{{ErrorType: "application", ErrorTag: errTag, ErrorSeverity: "error", ErrorMessage: escapedErrMsg}},
	}
	return marshalToXMLCommonSpeed(reply, frameEnd)
}
