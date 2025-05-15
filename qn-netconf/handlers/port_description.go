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

const PortDescriptionNamespace = "yang:get_port_description"
const PortDescriptionCapability = "yang:get_port_description" // Same as namespace for capability advertisement

// --- NETCONF Request Parsing Structures ---

// GetPortDescriptionRequestFilter is used to unmarshal the content of <port-description> from the filter.
type GetPortDescriptionRequestFilter struct {
	XMLName         xml.Name `xml:"port-description"` // Expects <port-description>
	Xmlns           string   `xml:"xmlns,attr,omitempty"`
	InterfaceNumber int      `xml:"interface-number"`
}

// GetFilterPayloadDesc is used to unmarshal the <filter> element for port description.
type GetFilterPayloadDesc struct {
	XMLName xml.Name                        `xml:"filter"`
	Content GetPortDescriptionRequestFilter `xml:"port-description"` // Expects <port-description> inside <filter>
}

// GetPayloadDesc is used to unmarshal the <get> element for port description.
type GetPayloadDesc struct {
	XMLName         xml.Name                         `xml:"get"`
	Filter          *GetFilterPayloadDesc            `xml:"filter,omitempty"`           // For <get><filter><port-description>...</port-description></filter></get>
	PortDescription *GetPortDescriptionRequestFilter `xml:"port-description,omitempty"` // For <get><port-description>...</port-description></get>
}

// FullRpcRequestForPortDescription is used to unmarshal the entire <rpc> request.
type FullRpcRequestForPortDescription struct {
	XMLName xml.Name       `xml:"rpc"`
	Get     GetPayloadDesc `xml:"get"`
}

// --- NETCONF Response XML Structures ---

// PortDescriptionPayload is the main data structure for the <port-description> element in the response.
type PortDescriptionPayload struct {
	XMLName         xml.Name `xml:"port-description"` // Tag name for the XML element
	Xmlns           string   `xml:"xmlns,attr"`       // Namespace attribute for <port-description>
	InterfaceNumber int      `xml:"interface-number"` // <interface-number>
	Description     string   `xml:"description"`      // <description>
}

// RpcReplyPortDescriptionGet is the top-level <rpc-reply> for the get port description operation.
type RpcReplyPortDescriptionGet struct {
	XMLName   xml.Name  `xml:"rpc-reply"`
	MessageID string    `xml:"-"` // Omit the MessageID field
	Data      *struct { // Anonymous struct for the <data> element
		XMLName                xml.Name `xml:"data"`
		PortDescriptionPayload PortDescriptionPayload
	} `xml:"data,omitempty"`
	Errors []RPCError `xml:"rpc-error,omitempty"` // Assumes RPCError is defined in package scope (e.g. from vlan.go)
}

// --- Miyagi JSON RPC Structs ---
type InterfaceDescriptionArg struct {
	InterfaceNumber int `json:"interface_number"`
}

var miyagiRequestCounterDesc uint32 // For local ID generation, suffixed to avoid conflict if in same package and not truly shared

// HandleGetPortDescription processes a NETCONF <get> request for port description.
func HandleGetPortDescription(miyagiSocketPath string, requestXML []byte, msgID string, frameEnd string) []byte {
	var parsedReq FullRpcRequestForPortDescription
	if err := xml.Unmarshal(requestXML, &parsedReq); err != nil {
		log.Printf("NETCONF_PORT_DESC_HANDLER: Error unmarshalling request: %v. Request: %s", err, string(requestXML))
		return buildErrorResponseBytesCommonDesc(msgID, "malformed-message", "Invalid request format", frameEnd)
	}

	var portDescFilterCriteria *GetPortDescriptionRequestFilter

	if parsedReq.Get.PortDescription != nil && parsedReq.Get.PortDescription.XMLName.Local == "port-description" {
		portDescFilterCriteria = parsedReq.Get.PortDescription
	} else if parsedReq.Get.Filter != nil && parsedReq.Get.Filter.Content.XMLName.Local == "port-description" {
		portDescFilterCriteria = &parsedReq.Get.Filter.Content
	} else {
		log.Printf("NETCONF_PORT_DESC_HANDLER: Malformed request, <port-description> filter not found. Request: %s", string(requestXML))
		return buildErrorResponseBytesCommonDesc(msgID, "malformed-message", "Missing or malformed <port-description> filter criteria", frameEnd)
	}

	if portDescFilterCriteria.Xmlns != PortDescriptionNamespace {
		log.Printf("NETCONF_PORT_DESC_HANDLER: Namespace mismatch in <port-description> filter. Expected '%s', got '%s'", PortDescriptionNamespace, portDescFilterCriteria.Xmlns)
	}

	interfaceNum := portDescFilterCriteria.InterfaceNumber
	if interfaceNum <= 0 {
		log.Printf("NETCONF_PORT_DESC_HANDLER: Invalid interface number %d from filter.", interfaceNum)
		return buildErrorResponseBytesCommonDesc(msgID, "invalid-value", "Interface number must be positive", frameEnd)
	}

	log.Printf("NETCONF_PORT_DESC_HANDLER: Querying description for interface number: %d", interfaceNum)

	miyagiReqPayload := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.Interface.Description",
			"arg": InterfaceDescriptionArg{InterfaceNumber: interfaceNum},
		},
		ID: int(generateMiyagiIDDesc()),
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReqPayload)
	if err != nil || miyagiResp.Error != nil {
		errMsg := "Error communicating with device agent for port description"
		if err != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, err)
		} else {
			errMsg = fmt.Sprintf("Device error for port description: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		}
		log.Printf("NETCONF_PORT_DESC_HANDLER: %s", errMsg)
		return buildErrorResponseBytesCommonDesc(msgID, "operation-failed", errMsg, frameEnd)
	}

	if miyagiResp.Result == nil {
		log.Printf("NETCONF_PORT_DESC_HANDLER: Miyagi returned no result for interface %d description.", interfaceNum)
		return buildErrorResponseBytesCommonDesc(msgID, "operation-failed", "No result from device agent for port description", frameEnd)
	}

	var description string
	if err := json.Unmarshal(miyagiResp.Result, &description); err != nil {
		log.Printf("NETCONF_PORT_DESC_HANDLER: Error unmarshalling Miyagi result for interface %d description: %v. Raw: %s", interfaceNum, err, string(miyagiResp.Result))
		return buildErrorResponseBytesCommonDesc(msgID, "operation-failed", "Invalid data format from device agent for port description", frameEnd)
	}

	responsePayload := PortDescriptionPayload{
		Xmlns:           PortDescriptionNamespace,
		InterfaceNumber: interfaceNum,
		Description:     description,
	}

	reply := RpcReplyPortDescriptionGet{
		Data: &struct {
			XMLName                xml.Name `xml:"data"`
			PortDescriptionPayload PortDescriptionPayload
		}{PortDescriptionPayload: responsePayload},
	}

	return marshalToXMLCommonDesc(reply, frameEnd)
}

func generateMiyagiIDDesc() uint32 {
	return atomic.AddUint32(&miyagiRequestCounterDesc, 1)
}

func marshalToXMLCommonDesc(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_PORT_DESC_HANDLER: FATAL: Failed to marshal XML: %v", err)
		return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rpc-reply><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`, frameEnd))
	}
	return append([]byte(xml.Header), append(append(xmlBytes, '\n'), []byte(frameEnd)...)...)
}

func buildErrorResponseBytesCommonDesc(msgID, errTag, errMsg, frameEnd string) []byte {
	escapedErrMsg := html.EscapeString(errMsg)
	reply := RpcReplyPortDescriptionGet{
		Errors: []RPCError{{ErrorType: "application", ErrorTag: errTag, ErrorSeverity: "error", ErrorMessage: escapedErrMsg}},
	}
	return marshalToXMLCommonDesc(reply, frameEnd)
}
