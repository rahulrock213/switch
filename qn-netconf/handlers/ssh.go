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

const SshConfigNamespace = "urn:example:params:xml:ns:yang:ssh-server-config"
const NetconfBaseNamespaceSSH = "urn:ietf:params:xml:ns:netconf:base:1.0"

// --- Common NETCONF XML Data Structures (for SSH handler) ---

type RpcReplySSH struct {
	// Changed XMLName to remove the base NETCONF namespace from the marshalled output.
	XMLName xml.Name `xml:"rpc-reply"`
	// MessageID field removed to prevent its output.
	// Data wrapper removed; SshConfig will be directly under rpc-reply.
	SshConfig *SshServerConfigData `xml:"ssh-server-config,omitempty"`
	Ok        *OkSSH               `xml:"ok,omitempty"`
	Errors    []RPCErrorSSH        `xml:"rpc-error,omitempty"`
}

type RPCErrorSSH struct {
	XMLName       xml.Name `xml:"rpc-error"`
	ErrorType     string   `xml:"error-type"`
	ErrorTag      string   `xml:"error-tag"`
	ErrorSeverity string   `xml:"error-severity"`
	ErrorMessage  string   `xml:"error-message"`
}

// OkSSH represents the <ok/> element.
type OkSSH struct {
	XMLName xml.Name `xml:"ok"`
}

// --- SSH Specific XML Data Structures ---

// SshServerConfigData is used for <ssh-server-config> in <data> or <config>
type SshServerConfigData struct {
	// Make XMLName namespace-agnostic here to allow unmarshalling from requests
	// that use a different namespace on the <ssh-server-config> tag (like "yang:set_ssh").
	XMLName xml.Name `xml:"ssh-server-config"`
	Enabled *bool    `xml:"enabled,omitempty"` // Use pointer to distinguish not present vs. explicit false
}

// EditConfigSshPayload is used to parse the <config> part of an <edit-config> for SSH
type EditConfigSshPayload struct {
	XMLName   xml.Name             `xml:"config"`
	SshConfig *SshServerConfigData `xml:"ssh-server-config"`
}

// --- Handler Functions ---

// HandleSSHGetConfig handles <get> or <get-config> for SSH status
func HandleSSHGetConfig(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.SSH.Server.Enabled",
			"arg": nil,
		},
		ID: 3, // Using a static ID, similar to interface.go
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_SSH_HANDLER: Error calling Miyagi for Get.SSH.Server.Enabled: %v", err)
		return buildErrorResponseBytesSSH(msgID, "application", "operation-failed", "Error communicating with device agent", frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error retrieving SSH status: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_SSH_HANDLER: Miyagi returned error: %s", errMsg)
		return buildErrorResponseBytesSSH(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	var miyagiStatusInt int // Expecting a raw integer (1 or 2)
	if err := json.Unmarshal(miyagiResp.Result, &miyagiStatusInt); err != nil {
		log.Printf("NETCONF_SSH_HANDLER: Error unmarshalling Miyagi response for SSH status: %v. Raw: %s", err, string(miyagiResp.Result))
		return buildErrorResponseBytesSSH(msgID, "application", "operation-failed", "Failed to parse SSH status from device", frameEnd)
	}

	// Convert Miyagi's integer result to boolean for NETCONF
	sshEnabled := miyagiStatusInt == 1

	// Create the actual data structure to be marshalled
	sshConfigPayload := SshServerConfigData{
		// For the GET response, explicitly set the desired XMLName with namespace
		XMLName: xml.Name{Space: "yang:ssh", Local: "ssh-server-config"},
		Enabled: &sshEnabled,
	}

	reply := RpcReplySSH{
		// MessageID is no longer part of RpcReplySSH for this simplified response
		SshConfig: &sshConfigPayload,
	}
	return marshalToXMLSSH(reply, frameEnd)
}

// HandleSSHEditConfig handles <edit-config> for SSH enable/disable
func HandleSSHEditConfig(miyagiSocketPath string, request []byte, msgID, frameEnd string) []byte {
	var editReq EditConfigSshPayload
	// We need to unmarshal from the <config> element.
	// A simple way is to find the <config> part of the request.
	configStartIndex := bytes.Index(request, []byte("<config>"))
	configEndIndex := bytes.LastIndex(request, []byte("</config>"))

	if configStartIndex == -1 || configEndIndex == -1 || configStartIndex >= configEndIndex {
		log.Printf("NETCONF_SSH_HANDLER: Malformed <edit-config> request, <config> tag not found or invalid: %s", string(request))
		return buildErrorResponseBytesSSH(msgID, "protocol", "malformed-message", "Malformed <edit-config> request", frameEnd)
	}
	configPayload := request[configStartIndex : configEndIndex+len("</config>")]

	log.Printf("NETCONF_SSH_HANDLER: DEBUG: Attempting to unmarshal configPayload: %s", string(configPayload))
	if err := xml.Unmarshal(configPayload, &editReq); err != nil {
		log.Printf("NETCONF_SSH_HANDLER: Error unmarshalling SSH <edit-config> payload: %v. Payload: %s", err, string(configPayload))
		return buildErrorResponseBytesSSH(msgID, "protocol", "malformed-message", "Invalid SSH configuration format", frameEnd)
	}

	if editReq.SshConfig == nil || editReq.SshConfig.Enabled == nil {
		log.Printf("NETCONF_SSH_HANDLER: Malformed SSH <edit-config>, <ssh-server-config><enabled> missing.")
		return buildErrorResponseBytesSSH(msgID, "protocol", "missing-element", "<ssh-server-config><enabled> is required", frameEnd)
	}

	var miyagiUID string
	if *editReq.SshConfig.Enabled {
		miyagiUID = "Agent.Switch.Set.SSH.Enable"
	} else {
		miyagiUID = "Agent.Switch.Set.SSH.Disable"
	}

	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{"uid": miyagiUID, "arg": nil},
		ID:     4, // Using a static ID, ensure it's different if Miyagi needs unique IDs per call
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_SSH_HANDLER: Error calling Miyagi for %s: %v", miyagiUID, err)
		return buildErrorResponseBytesSSH(msgID, "application", "operation-failed", "Error communicating with device agent to set SSH status", frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error setting SSH status: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_SSH_HANDLER: Miyagi returned error for %s: %s", miyagiUID, errMsg)
		return buildErrorResponseBytesSSH(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	// If Miyagi call is successful
	reply := RpcReplySSH{
		// MessageID is no longer part of RpcReplySSH
		Ok: &OkSSH{},
	}
	return marshalToXMLSSH(reply, frameEnd)
}

// --- Helper Functions (specific to SSH handler to avoid conflicts if they diverge) ---

func marshalToXMLSSH(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_SSH_HANDLER: FATAL: Failed to marshal XML: %v", err)
		// Fallback error
		return []byte(fmt.Sprintf(
			`<?xml version="1.0" encoding="UTF-8"?><rpc-reply xmlns="%s"><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`,
			NetconfBaseNamespaceSSH, frameEnd,
		))
	}
	return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
}

func buildErrorResponseBytesSSH(msgID, errType, errTag, errMsg, frameEnd string) []byte {
	// Basic XML escaping for the error message
	escapedErrMsg := strings.ReplaceAll(errMsg, "<", "&lt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, ">", "&gt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, "&", "&amp;")

	reply := RpcReplySSH{
		Errors: []RPCErrorSSH{
			{
				ErrorType:     errType,
				ErrorTag:      errTag,
				ErrorSeverity: "error",
				ErrorMessage:  escapedErrMsg,
			},
		},
	}
	return marshalToXMLSSH(reply, frameEnd)
}
