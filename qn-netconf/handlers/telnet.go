package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"strings"

	"qn-netconf/miyagi" // Assuming your miyagi client is in "qn-netconf/miyagi"
)

const TelnetConfigNamespace = "urn:example:params:xml:ns:yang:telnet-server-config"
const NetconfBaseNamespaceTelnet = "urn:ietf:params:xml:ns:netconf:base:1.0"

// --- Common NETCONF XML Data Structures (for Telnet handler) ---

type RpcReplyTelnet struct {
	XMLName   xml.Name         `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
	MessageID string           `xml:"message-id,attr"`
	Data      *DataTelnet      `xml:"data,omitempty"`
	Ok        *OkTelnet        `xml:"ok,omitempty"`
	Errors    []RPCErrorTelnet `xml:"rpc-error,omitempty"`
}

type OkTelnet struct {
	XMLName xml.Name `xml:"ok"`
}

type DataTelnet struct {
	XMLName      xml.Name                `xml:"data"`
	TelnetConfig *TelnetServerConfigData `xml:"telnet-server-config,omitempty"`
}

type RPCErrorTelnet struct {
	XMLName       xml.Name `xml:"rpc-error"`
	ErrorType     string   `xml:"error-type"`
	ErrorTag      string   `xml:"error-tag"`
	ErrorSeverity string   `xml:"error-severity"`
	ErrorMessage  string   `xml:"error-message"`
}

// --- Telnet Specific XML Data Structures ---

// TelnetServerConfigData is used for <telnet-server-config> in <data> or <config>
type TelnetServerConfigData struct {
	XMLName xml.Name `xml:"telnet-server-config"`
	Xmlns   string   `xml:"xmlns,attr,omitempty"`
	Enabled *bool    `xml:"enabled,omitempty"` // Use pointer to distinguish not present vs. explicit false
}

// EditConfigTelnetPayload is used to parse the <config> part of an <edit-config> for Telnet
type EditConfigTelnetPayload struct {
	XMLName      xml.Name                `xml:"config"`
	TelnetConfig *TelnetServerConfigData `xml:"telnet-server-config"`
}

// --- Handler Functions ---

// HandleTelnetGetConfig handles <get> or <get-config> for Telnet status
func HandleTelnetGetConfig(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.Telnet.Enabled",
			"arg": nil,
		},
		ID: 5, // Using a static ID
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_TELNET_HANDLER: Error calling Miyagi for Get.Telnet.Enabled: %v", err)
		return buildErrorResponseBytesTelnet(msgID, "application", "operation-failed", "Error communicating with device agent", frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error retrieving Telnet status: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_TELNET_HANDLER: Miyagi returned error: %s", errMsg)
		return buildErrorResponseBytesTelnet(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	// Assuming Miyagi returns a raw integer (1 for enabled, 2 for disabled) similar to SSH
	var miyagiStatusInt int
	if err := json.Unmarshal(miyagiResp.Result, &miyagiStatusInt); err != nil {
		log.Printf("NETCONF_TELNET_HANDLER: Error unmarshalling Miyagi response for Telnet status: %v. Raw: %s", err, string(miyagiResp.Result))
		return buildErrorResponseBytesTelnet(msgID, "application", "operation-failed", "Failed to parse Telnet status from device", frameEnd)
	}

	telnetEnabled := miyagiStatusInt == 1

	telnetConfigPayload := TelnetServerConfigData{
		Xmlns:   TelnetConfigNamespace,
		Enabled: &telnetEnabled,
	}

	reply := RpcReplyTelnet{
		MessageID: msgID,
		Data: &DataTelnet{
			TelnetConfig: &telnetConfigPayload,
		},
	}
	return marshalToXMLTelnet(reply, frameEnd)
}

// HandleTelnetEditConfig handles <edit-config> for Telnet enable/disable
func HandleTelnetEditConfig(miyagiSocketPath string, request []byte, msgID, frameEnd string) []byte {
	var editReq EditConfigTelnetPayload
	configStartIndex := bytes.Index(request, []byte("<config>"))
	configEndIndex := bytes.LastIndex(request, []byte("</config>"))

	if configStartIndex == -1 || configEndIndex == -1 || configStartIndex >= configEndIndex {
		log.Printf("NETCONF_TELNET_HANDLER: Malformed <edit-config> request, <config> tag not found or invalid: %s", string(request))
		return buildErrorResponseBytesTelnet(msgID, "protocol", "malformed-message", "Malformed <edit-config> request", frameEnd)
	}
	configPayload := request[configStartIndex : configEndIndex+len("</config>")]

	if err := xml.Unmarshal(configPayload, &editReq); err != nil {
		log.Printf("NETCONF_TELNET_HANDLER: Error unmarshalling Telnet <edit-config> payload: %v. Payload: %s", err, string(configPayload))
		return buildErrorResponseBytesTelnet(msgID, "protocol", "malformed-message", "Invalid Telnet configuration format", frameEnd)
	}

	if editReq.TelnetConfig == nil || editReq.TelnetConfig.Enabled == nil {
		log.Printf("NETCONF_TELNET_HANDLER: Malformed Telnet <edit-config>, <telnet-server-config><enabled> missing.")
		return buildErrorResponseBytesTelnet(msgID, "protocol", "missing-element", "<telnet-server-config><enabled> is required", frameEnd)
	}

	var miyagiUID string
	if *editReq.TelnetConfig.Enabled {
		miyagiUID = "Agent.Switch.Set.TelnetServerEnable"
	} else {
		miyagiUID = "Agent.Switch.Set.TelnetServerDisable"
	}

	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{"uid": miyagiUID, "arg": nil},
		ID:     6, // Using a static ID
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_TELNET_HANDLER: Error calling Miyagi for %s: %v", miyagiUID, err)
		return buildErrorResponseBytesTelnet(msgID, "application", "operation-failed", "Error communicating with device agent to set Telnet status", frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error setting Telnet status: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_TELNET_HANDLER: Miyagi returned error for %s: %s", miyagiUID, errMsg)
		return buildErrorResponseBytesTelnet(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	reply := RpcReplyTelnet{MessageID: msgID, Ok: &OkTelnet{}}
	return marshalToXMLTelnet(reply, frameEnd)
}

// --- Helper Functions (specific to Telnet handler) ---

func marshalToXMLTelnet(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_TELNET_HANDLER: FATAL: Failed to marshal XML: %v", err)
		return []byte(fmt.Sprintf(
			`<?xml version="1.0" encoding="UTF-8"?><rpc-reply xmlns="%s"><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`,
			NetconfBaseNamespaceTelnet, frameEnd,
		))
	}
	return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
}

func buildErrorResponseBytesTelnet(msgID, errType, errTag, errMsg, frameEnd string) []byte {
	escapedErrMsg := strings.ReplaceAll(errMsg, "<", "&lt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, ">", "&gt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, "&", "&amp;")

	reply := RpcReplyTelnet{
		MessageID: msgID,
		Errors: []RPCErrorTelnet{
			{
				ErrorType:     errType,
				ErrorTag:      errTag,
				ErrorSeverity: "error",
				ErrorMessage:  escapedErrMsg,
			},
		},
	}
	return marshalToXMLTelnet(reply, frameEnd)
}
