package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"strings"

	"qn-netconf/miyagi"
)

const StpGlobalConfigNamespace = "yang:stp-global-config"
const NetconfBaseNamespaceStp = "urn:ietf:params:xml:ns:netconf:base:1.0"

// --- Common NETCONF XML Data Structures ---
type RpcReplyStp struct {
	XMLName         xml.Name             `xml:"rpc-reply"`                   // Simplified root
	StpGlobalConfig *StpGlobalConfigData `xml:"stp-global-config,omitempty"` // For GET response
	Result          string               `xml:"result,omitempty"`            // For edit-config response
	Errors          []RPCErrorStp        `xml:"rpc-error,omitempty"`
}

type RPCErrorStp struct {
	XMLName       xml.Name `xml:"rpc-error"`
	ErrorType     string   `xml:"error-type"`
	ErrorTag      string   `xml:"error-tag"`
	ErrorSeverity string   `xml:"error-severity"`
	ErrorMessage  string   `xml:"error-message"`
}

// --- STP Specific XML Data Structures ---
type StpGlobalConfigData struct {
	// XMLName made namespace-agnostic for flexible unmarshalling in edit-config.
	// Namespace will be explicitly set when marshalling for GET response.
	XMLName xml.Name `xml:"stp-global-config"`
	Enabled *bool    `xml:"enabled,omitempty"`
}

// EditConfigStpPayload is used to parse the <config> part of an <edit-config> for STP
type EditConfigStpPayload struct {
	XMLName         xml.Name             `xml:"config"`
	StpGlobalConfig *StpGlobalConfigData `xml:"stp-global-config"`
}

// --- Handler Functions ---

// HandleStpGetConfig handles <get-config> for global STP status
func HandleStpGetConfig(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{"uid": "Agent.Switch.Get.STP.Enabled", "arg": nil},
		ID:     13, // Unique ID
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil || miyagiResp.Error != nil {
		errMsg := "Error communicating with device agent for STP status"
		if err != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, err)
		} else {
			errMsg = fmt.Sprintf("Device error for STP status: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		}
		log.Printf("NETCONF_STP_HANDLER: %s", errMsg)
		return buildErrorResponseBytesStp(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	var miyagiStatusInt int // Assuming Miyagi returns 1 for enabled, 2 for disabled (or other non-1)
	if err := json.Unmarshal(miyagiResp.Result, &miyagiStatusInt); err != nil {
		log.Printf("NETCONF_STP_HANDLER: Error unmarshalling Miyagi response for STP status: %v. Raw: %s", err, string(miyagiResp.Result))
		return buildErrorResponseBytesStp(msgID, "application", "operation-failed", "Failed to parse STP status from device", frameEnd)
	}

	stpEnabled := miyagiStatusInt == 1

	stpData := StpGlobalConfigData{
		// Explicitly set XMLName with desired namespace for GET response
		XMLName: xml.Name{Space: "yang:stp", Local: "stp-global-config"},
		Enabled: &stpEnabled,
	}

	reply := RpcReplyStp{
		StpGlobalConfig: &stpData, // Directly embed
	}
	return marshalToXMLStp(reply, frameEnd)
}

// HandleStpEditConfig handles <edit-config> for global STP enable/disable
func HandleStpEditConfig(miyagiSocketPath string, request []byte, msgID, frameEnd string) []byte {
	var editReq EditConfigStpPayload
	configStartIndex := bytes.Index(request, []byte("<config>"))
	configEndIndex := bytes.LastIndex(request, []byte("</config>"))
	if configStartIndex == -1 || configEndIndex == -1 || configStartIndex >= configEndIndex {
		return buildErrorResponseBytesStp(msgID, "protocol", "malformed-message", "Malformed <edit-config> request", frameEnd)
	}
	configPayload := request[configStartIndex : configEndIndex+len("</config>")]

	if err := xml.Unmarshal(configPayload, &editReq); err != nil {
		log.Printf("NETCONF_STP_HANDLER: Error unmarshalling STP <edit-config> payload: %v. Payload: %s", err, string(configPayload))
		return buildErrorResponseBytesStp(msgID, "protocol", "malformed-message", "Invalid STP configuration format", frameEnd)
	}

	if editReq.StpGlobalConfig == nil || editReq.StpGlobalConfig.Enabled == nil {
		return buildErrorResponseBytesStp(msgID, "protocol", "missing-element", "<stp-global-config><enabled> is required", frameEnd)
	}

	var miyagiUID string
	if *editReq.StpGlobalConfig.Enabled {
		miyagiUID = "Agent.Switch.Set.STP.Enable"
	} else {
		miyagiUID = "Agent.Switch.Set.STP.Disable"
	}

	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{"uid": miyagiUID, "arg": nil},
		ID:     14, // Unique ID
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil || miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Error communicating with device agent to set STP status (UID: %s)", miyagiUID)
		if err != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, err)
		} else {
			errMsg = fmt.Sprintf("Device error setting STP status (UID: %s): %s (code: %d)", miyagiUID, miyagiResp.Error.Message, miyagiResp.Error.Code)
		}
		log.Printf("NETCONF_STP_HANDLER: %s", errMsg)
		return buildErrorResponseBytesStp(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	reply := RpcReplyStp{
		// MessageID is no longer part of RpcReplyStp
		Result: "ok",
	}
	return marshalToXMLStp(reply, frameEnd)
}

// --- Helper Functions ---
func marshalToXMLStp(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_STP_HANDLER: FATAL: Failed to marshal XML: %v", err)
		return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rpc-reply xmlns="%s"><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`, NetconfBaseNamespaceStp, frameEnd))
	}
	return append([]byte(xml.Header), append(append(xmlBytes, '\n'), []byte(frameEnd)...)...)
}

func buildErrorResponseBytesStp(msgID, errType, errTag, errMsg, frameEnd string) []byte {
	escapedErrMsg := strings.ReplaceAll(errMsg, "<", "&lt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, ">", "&gt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, "&", "&amp;")
	reply := RpcReplyStp{
		// MessageID is no longer part of RpcReplyStp
		Errors: []RPCErrorStp{{ErrorType: errType, ErrorTag: errTag, ErrorSeverity: "error", ErrorMessage: escapedErrMsg}},
	}
	return marshalToXMLStp(reply, frameEnd)
}
