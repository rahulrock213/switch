package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"sort"
	"strings"

	"qn-netconf/miyagi"
)

const IpInterfaceNamespace = "yang:ip-interface"
const NetconfBaseNamespaceIpInterface = "urn:ietf:params:xml:ns:netconf:base:1.0"
const IpInterfaceCapability = "yang:ip-interface" // Consistent capability format

// --- Common NETCONF XML Data Structures ---

type RpcReplyIpInterface struct {
	// Simplified RpcReply for edit-config responses
	XMLName xml.Name `xml:"rpc-reply"`
	// MessageID removed
	// Data field removed as GET is custom and edit-config only needs Ok/Error
	Result string                `xml:"result,omitempty"` // Changed from Ok to Result
	Errors []RPCErrorIpInterface `xml:"rpc-error,omitempty"`
}

// DataIpInterface struct is no longer used for the simplified GET/edit-config paths.
// If it were used for a standard <get-config> response, it would be defined here.

type RPCErrorIpInterface struct {
	XMLName       xml.Name `xml:"rpc-error"`
	ErrorType     string   `xml:"error-type"`
	ErrorTag      string   `xml:"error-tag"`
	ErrorSeverity string   `xml:"error-severity"`
	ErrorMessage  string   `xml:"error-message"`
}

// --- IP Interface Specific XML Data Structures (NETCONF Style - primarily for EditConfig) ---

// IpInterfacesData is the container for multiple IP interface configurations
type IpInterfacesData struct {
	// XMLName made namespace-agnostic for flexible unmarshalling in edit-config.
	XMLName xml.Name `xml:"ip-interfaces"` // For edit-config unmarshalling
	// Xmlns attribute removed, namespace handled by request content
	Interfaces []IpInterfaceData `xml:"interface"`
}

// IpInterfaceData represents a single IP interface's configuration
type IpInterfaceData struct {
	XMLName    xml.Name `xml:"interface"`
	Operation  string   `xml:"operation,attr,omitempty"` // For edit-config: create, delete, merge
	Name       string   `xml:"name"`                     // e.g., vlan1, loopback0, te1/0/1
	IpAddress  string   `xml:"ip-address,omitempty"`
	MaskPrefix string   `xml:"mask-prefix,omitempty"` // e.g., 255.255.255.0 or /24
	// Add other IP related fields if necessary, e.g., DHCP, secondary IPs etc.
}

// --- Miyagi JSON Data Structures ---
type MiyagiIpType struct {
	Value       int    `json:"value"`
	Description string `json:"description"`
}

type MiyagiIpDetail struct {
	IpAddress  string        `json:"ip4"`
	SubnetMask string        `json:"subnet_mask"` // Changed from MaskPrefix
	Type       *MiyagiIpType `json:"type,omitempty"`
	IfIndex    *int          `json:"ifindex,omitempty"` // Use pointer to handle potential null/missing
}

// --- Custom XML Output Structures for Get IP Interface ---
type XmlResultRootIp struct {
	// Changed root tag for custom GET response to rpc-reply
	XMLName    xml.Name              `xml:"rpc-reply"`
	Interfaces []XmlIpInterfaceEntry `xml:",innerxml"` // Custom marshalling for dynamic tags
}

type XmlIpInterfaceEntry struct {
	XMLName    xml.Name   // Dynamic: e.g., <1>, <te1_0_1>
	Ip4        string     `xml:"ip4"`
	SubnetMask string     `xml:"subnet_mask"`
	Type       *XmlIpType `xml:"type,omitempty"`
	IfIndex    *int       `xml:"ifindex,omitempty"` // Pointer to omit if nil
}

type XmlIpType struct {
	Value       int    `xml:"value"`
	Description string `xml:"description"`
}

// Custom marshaller for XmlResultRootIp
func (r XmlResultRootIp) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, iface := range r.Interfaces {
		if err := e.EncodeElement(iface, xml.StartElement{Name: iface.XMLName}); err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// marshalIpInterfaceEntries is a helper to marshal only the list of XmlIpInterfaceEntry
// using the custom logic for dynamic tags.
func marshalIpInterfaceEntries(interfaces []XmlIpInterfaceEntry, prefix string, indent string) ([]byte, error) {
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	// Always apply indentation. The prefix is for each line generated by this encoder,
	// and indent is for subsequent levels of nesting within elements encoded here.
	encoder.Indent(prefix, indent)
	for _, iface := range interfaces {
		// Encode each interface using its dynamic name (iface.XMLName)
		if err := encoder.EncodeElement(iface, xml.StartElement{Name: iface.XMLName}); err != nil {
			return nil, fmt.Errorf("failed to encode IP interface %s: %w", iface.XMLName.Local, err)
		}
	}
	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush encoder for IP interface entries: %w", err)
	}
	return buf.Bytes(), nil
}

// --- Handler Functions ---

// HandleIpInterfaceGetConfig handles <get> or <get-config> for IP interface status
func HandleIpInterfaceGetConfig(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.Interface.Table", // As per reference
			"arg": nil,
		},
		ID: 9, // Static ID
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_IP_IF_HANDLER: Error calling Miyagi for Get.Interface.Table: %v", err)
		return buildErrorResponseBytesIpInterface(msgID, "application", "operation-failed", "Error communicating with device agent", frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error retrieving IP interface table: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_IP_IF_HANDLER: Miyagi returned error: %s", errMsg)
		return buildErrorResponseBytesIpInterface(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	// Miyagi returns a map where keys are interface names and values are their details.
	var miyagiInterfaceTable map[string]MiyagiIpDetail // Using a simplified struct for IP details
	if err := json.Unmarshal(miyagiResp.Result, &miyagiInterfaceTable); err != nil {
		log.Printf("NETCONF_IP_IF_HANDLER: Error unmarshalling Miyagi IP interface table for custom XML: %v. Raw: %s", err, string(miyagiResp.Result))
		// For custom XML, we might return a simpler error or an empty <result/>
		return []byte(fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?><error>Failed to parse IP interface data from device: %v. Raw: %s</error>", err, string(miyagiResp.Result)))
	}

	var xmlIpEntries []XmlIpInterfaceEntry
	var interfaceNames []string
	for name := range miyagiInterfaceTable {
		interfaceNames = append(interfaceNames, name)
	}
	sort.Strings(interfaceNames) // Sort for consistent output

	sanitizer := strings.NewReplacer(" ", "_", "/", "_", ":", "_")

	for _, name := range interfaceNames {
		details := miyagiInterfaceTable[name]

		// Sanitize name for XML tag
		tagName := sanitizer.Replace(name)
		if tagName == "" { // Should not happen if name is valid
			tagName = "unknown_interface"
		}

		entry := XmlIpInterfaceEntry{
			XMLName:    xml.Name{Local: tagName},
			Ip4:        details.IpAddress,  // Renamed from IpAddress
			SubnetMask: details.SubnetMask, // Renamed from MaskPrefix
			IfIndex:    details.IfIndex,
		}
		if details.Type != nil {
			entry.Type = &XmlIpType{
				Value:       details.Type.Value,
				Description: details.Type.Description,
			}
		}
		xmlIpEntries = append(xmlIpEntries, entry)
	}

	// --- Direct Construction of XML with <rpc-reply> root ---
	// Marshal the entries themselves.
	// The prefix "  " for encoder.Indent will ensure each entry (like <1>, <MTPLAP...>) starts indented by two spaces.
	// The indent "  " for encoder.Indent will ensure children of these entries are further indented by two spaces.
	// This prepares entriesIndentedBytes to be placed inside an <ip-interfaces> tag that itself is indented.
	entriesIndentedBytes, err := marshalIpInterfaceEntries(xmlIpEntries, "    ", "  ") // Prefix "    " for 4-space indent, children indent by "  "
	if err != nil {
		log.Printf("NETCONF_IP_IF_HANDLER: Error marshalling inner IP interface XML: %v", err)
		// It's better to use the structured error response builder
		return buildErrorResponseBytesIpInterface(msgID, "application", "internal-error", fmt.Sprintf("Error generating XML for IP interfaces: %v", err), frameEnd)
	}

	var fullResponse bytes.Buffer
	fullResponse.WriteString(xml.Header)

	fullResponse.WriteString("<rpc-reply>\n") // Manually write the root tag, followed by a newline

	fullResponse.WriteString("  <ip-interfaces>\n") // Manually write the container tag, indented and followed by a newline

	if len(entriesIndentedBytes) > 0 {
		// entriesIndentedBytes are already correctly indented (each line starts with "  ")
		// and typically end with a newline if not empty.
		fullResponse.Write(entriesIndentedBytes)
	}
	fullResponse.WriteString("\n")                   // Newline after XML header
	fullResponse.WriteString("  </ip-interfaces>\n") // Closing container tag, indented
	fullResponse.WriteString("</rpc-reply>\n")       // Closing root tag

	// Append a newline, then the frame end terminator, similar to route.go
	return append(fullResponse.Bytes(), []byte(frameEnd)...) // The last WriteString already added a \n
}

// HandleIpInterfaceEditConfig handles <edit-config> for setting IP interface properties
func HandleIpInterfaceEditConfig(miyagiSocketPath string, request []byte, msgID, frameEnd string) []byte {
	var editReq struct { // Anonymous struct to parse <config><ip-interfaces>...</ip-interfaces></config>
		XMLName          xml.Name         `xml:"config"`
		IpInterfacesData IpInterfacesData `xml:"ip-interfaces"`
	}

	configStartIndex := bytes.Index(request, []byte("<config>"))
	configEndIndex := bytes.LastIndex(request, []byte("</config>"))
	if configStartIndex == -1 || configEndIndex == -1 || configStartIndex >= configEndIndex {
		return buildErrorResponseBytesIpInterface(msgID, "protocol", "malformed-message", "Malformed <edit-config> request", frameEnd)
	}
	configPayload := request[configStartIndex : configEndIndex+len("</config>")]

	if err := xml.Unmarshal(configPayload, &editReq); err != nil {
		log.Printf("NETCONF_IP_IF_HANDLER: Error unmarshalling IP interface <edit-config> payload: %v. Payload: %s", err, string(configPayload))
		return buildErrorResponseBytesIpInterface(msgID, "protocol", "malformed-message", "Invalid IP interface configuration format", frameEnd)
	}

	if len(editReq.IpInterfacesData.Interfaces) == 0 {
		return buildErrorResponseBytesIpInterface(msgID, "protocol", "missing-element", "<interface> element is required under <ip-interfaces>", frameEnd)
	}

	// Process each interface configuration. For simplicity, this example processes one.
	// A real implementation might loop or handle multiple.
	for _, ifaceConfig := range editReq.IpInterfacesData.Interfaces {
		// For this example, we only support 'create' or 'merge' as setting an IP.
		// 'delete' would require a different Miyagi UID or logic.
		if ifaceConfig.Operation != "create" && ifaceConfig.Operation != "merge" && ifaceConfig.Operation != "" { // "" implies merge
			return buildErrorResponseBytesIpInterface(msgID, "protocol", "bad-attribute", fmt.Sprintf("Unsupported operation: %s for IP interface", ifaceConfig.Operation), frameEnd)
		}

		if ifaceConfig.Name == "" || ifaceConfig.IpAddress == "" || ifaceConfig.MaskPrefix == "" {
			return buildErrorResponseBytesIpInterface(msgID, "protocol", "missing-attribute", "Interface name, ip-address, and mask-prefix are required.", frameEnd)
		}

		miyagiArgs := map[string]interface{}{
			"interface_name": ifaceConfig.Name,
			"ip_address":     ifaceConfig.IpAddress,
			"mask_prefix":    ifaceConfig.MaskPrefix,
		}

		miyagiReq := miyagi.MiyagiRequest{
			Method: "call",
			Params: map[string]interface{}{"uid": "Agent.Switch.Set.IPv4Addressing.IpAddressDefine", "arg": miyagiArgs},
			ID:     10, // Static ID
		}

		miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
		if err != nil {
			log.Printf("NETCONF_IP_IF_HANDLER: Error calling Miyagi for Set.IPv4Addressing.IpAddressDefine: %v", err)
			return buildErrorResponseBytesIpInterface(msgID, "application", "operation-failed", "Error communicating with device agent to set IP interface", frameEnd)
		}

		if miyagiResp.Error != nil {
			errMsg := fmt.Sprintf("Device error setting IP interface %s: %s (code: %d)", ifaceConfig.Name, miyagiResp.Error.Message, miyagiResp.Error.Code)
			log.Printf("NETCONF_IP_IF_HANDLER: Miyagi returned error: %s", errMsg)
			return buildErrorResponseBytesIpInterface(msgID, "application", "operation-failed", errMsg, frameEnd)
		}
		// If processing multiple, continue. Stop on first error for simplicity.
	}

	reply := RpcReplyIpInterface{
		// MessageID is no longer part of RpcReplyIpInterface
		Result: "ok", // Changed to use Result
	}
	return marshalToXMLIpInterface(reply, frameEnd)
}

// --- Helper Functions ---

func marshalToXMLIpInterface(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_IP_IF_HANDLER: FATAL: Failed to marshal XML: %v", err)
		// Fallback error, ensuring it's also simple
		return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rpc-reply xmlns="%s"><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`, NetconfBaseNamespaceIpInterface, frameEnd))
	}
	return append([]byte(xml.Header), append(append(xmlBytes, '\n'), []byte(frameEnd)...)...)
}

func buildErrorResponseBytesIpInterface(msgID, errType, errTag, errMsg, frameEnd string) []byte {
	// msgID is kept for logging/internal tracking but won't be in the simplified XML error response
	escapedErrMsg := strings.ReplaceAll(errMsg, "<", "&lt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, ">", "&gt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, "&", "&amp;")
	reply := RpcReplyIpInterface{
		// MessageID is no longer part of RpcReplyIpInterface
		Errors: []RPCErrorIpInterface{{ErrorType: errType, ErrorTag: errTag, ErrorSeverity: "error", ErrorMessage: escapedErrMsg}},
	}
	return marshalToXMLIpInterface(reply, frameEnd)
}
