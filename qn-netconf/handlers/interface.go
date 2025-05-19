package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml" // Import encoding/xml
	"fmt"
	"log"
	"sort" // For sorting interfaces
	"strconv"
	"strings" // For prefix checking in sorting

	"qn-netconf/miyagi" // Assuming your miyagi client is in "net_conf/miyagi"
)

const InterfaceNamespace = "yang:interfaces"                             // Example namespace
const InterfaceCapability = "yang:interfaces"                            // Consistent capability format
const NetconfBaseNamespaceIF = "urn:ietf:params:xml:ns:netconf:base:1.0" // Using IF suffix for clarity if in same package scope as vlan.go without shared utils

// --- Common NETCONF XML Data Structures (Ideally in a shared package) ---
// These are similar to vlan.go. If vlan.go and interface.go are in the same package,
// these definitions would conflict. Ideally, they'd be in a shared 'netconfutil' package.
// For this example, I'm suffixing with 'IF' to make it self-contained for copy-pasting.

// RpcReplyIF is the generic NETCONF rpc-reply structure.
type RpcReplyIF struct {
	XMLName   xml.Name     `xml:"rpc-reply"`
	MessageID string       `xml:"message-id,attr"`
	Data      *DataIF      `xml:"data,omitempty"`
	Ok        *OkIF        `xml:"ok,omitempty"`
	Errors    []RPCErrorIF `xml:"rpc-error,omitempty"`
}

// OkIF represents the <ok/> element.
type OkIF struct {
	XMLName xml.Name `xml:"ok"`
}

// DataIF wraps the actual data payload.
type DataIF struct {
	XMLName        xml.Name    `xml:"data"`
	InterfacesData interface{} `xml:",innerxml"` // Use innerxml to embed pre-formatted or dynamic XML
}

// RPCErrorIF represents a NETCONF rpc-error.
type RPCErrorIF struct {
	XMLName       xml.Name `xml:"rpc-error"`
	ErrorType     string   `xml:"error-type"`
	ErrorTag      string   `xml:"error-tag"`
	ErrorSeverity string   `xml:"error-severity"`
	ErrorMessage  string   `xml:"error-message"`
}

// --- Custom XML Output Structures (as per user request) ---

// XmlRoot is the root element for the custom XML output for interface details.
// Its XMLName will define the root tag and its namespace.
type XmlRoot struct {
	XMLName xml.Name `xml:"rpc-reply"` // This should be the intended root tag
	// Interfaces are still ordered and marshalled directly under the root tag
	// and then marshalled directly under <root>
	Interfaces []XmlInterfaceElement
}

// XmlInterfaceElement holds the data for one interface, including its dynamic name.
type XmlInterfaceElement struct {
	XMLName         xml.Name              // This will be set dynamically to the interface name (e.g., "te1/0/1")
	IfDescription   string                `xml:"if_description"`
	IfIndex         string                `xml:"ifIndex"`       // Changed to string for empty tag ""
	IfType          string                `xml:"ifType"`        // Changed to string for empty tag ""
	IfSpeed         string                `xml:"ifSpeed"`       // Changed to string for empty tag ""
	IfAdminStatus   *XmlStatusDescription `xml:"ifAdminStatus"` // Pointer to allow empty <ifAdminStatus></ifAdminStatus>
	IfPhysAddress   string                `xml:"ifPhysAddress"` // omitempty removed if empty tag desired for empty string
	IfOperStatus    *XmlStatusDescription `xml:"ifOperStatus"`  // Pointer to allow empty <ifOperStatus></ifOperStatus>
	IfMtu           string                `xml:"ifMtu"`         // Changed to string for empty tag ""
	IfInOctets      string                `xml:"ifInOctets"`    // Changed to string for empty tag ""
	IfOutOctets     string                `xml:"ifOutOctets"`   // Changed to string for empty tag ""
	IfDuplex        *XmlStatusDescription `xml:"if_duplex"`     // XML tag is if_duplex
	PortMode        *XmlStatusDescription `xml:"port_mode"`
	NativeVlan      string                `xml:"native_vlan"`   // Changed to string for empty tag ""
	FlowControl     *string               `xml:"flow_control"`  // Pointer to produce <flow_control></flow_control> or <flow_control>value</flow_control>
	ComboMode       *string               `xml:"combo_mode"`    // Pointer to produce <combo_mode></combo_mode> for null
	Vlan            string                `xml:"vlans"`         // Changed to string for empty tag ""
	UntaggedVlanVal string                `xml:"untagged_vlan"` // Changed to string for empty tag ""
	TaggedVlan      string                `xml:"tagged_vlan"`   // Added, as string for empty tag ""
}

type XmlStatusDescription struct {
	Value       *int   `xml:"value,omitempty"`       // Pointer to omit if JSON value is null
	Description string `xml:"description,omitempty"` // Omitted if empty
}

// Custom marshaller for XmlRoot to handle dynamic interface tags directly under <root>
func (r XmlRoot) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	log.Printf("NETCONF_IF_HANDLER: MarshalXML called for XmlRoot. Intended root tag from start.Name.Local: %s", start.Name.Local)
	// start.Name.Local should be "rpc-reply" if XMLName tag is effective
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	for _, iface := range r.Interfaces {
		// Encode each interface using its dynamic name (iface.XMLName)
		if err := e.EncodeElement(iface, xml.StartElement{Name: iface.XMLName}); err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name}) // End <rpc-reply>
}

// --- Miyagi JSON Data Structures ---

type MiyagiStatusDescriptionJSON struct {
	Value       *int   `json:"value"` // Pointer to handle null from JSON
	Description string `json:"description"`
}

// MiyagiInterfaceDetail defines the structure of details for a single interface from Miyagi.
// Updated to match the full structure from Untitled-1.json
type MiyagiInterfaceDetail struct {
	IfDescription string                       `json:"if_description"`
	IfIndex       *int                         `json:"ifIndex"`
	IfType        *int                         `json:"ifType"`
	IfSpeed       *int                         `json:"ifSpeed"`
	IfAdminStatus *MiyagiStatusDescriptionJSON `json:"ifAdminStatus"`
	IfPhysAddress string                       `json:"ifPhysAddress"`
	IfOperStatus  *MiyagiStatusDescriptionJSON `json:"ifOperStatus"`
	IfMtu         *int                         `json:"ifMtu"`
	IfInOctets    *int64                       `json:"ifInOctets"`
	IfOutOctets   *int64                       `json:"ifOutOctets"`
	IfDuplex      *MiyagiStatusDescriptionJSON `json:"if_duplex"`
	PortMode      *MiyagiStatusDescriptionJSON `json:"port_mode"`
	NativeVlan    *int                         `json:"native_vlan"`
	FlowControl   *string                      `json:"flow_control"` // JSON "off" or null
	ComboMode     *string                      `json:"combo_mode"`   // JSON null
	Vlans         []int                        `json:"vlans"`
	UntaggedVlan  []int                        `json:"untagged_vlan"`
	TaggedVlan    []int                        `json:"tagged_vlan"`
}

// marshalInnerInterfaces is a helper to marshal only the list of XmlInterfaceElement
// using the custom logic for dynamic tags.
func marshalInnerInterfaces(interfaces []XmlInterfaceElement, prefix string, indent string) ([]byte, error) {
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	if prefix == "" && indent != "" { // Apply indent if specified
		encoder.Indent("", indent)
	}

	for _, iface := range interfaces {
		// Encode each interface using its dynamic name (iface.XMLName)
		if err := encoder.EncodeElement(iface, xml.StartElement{Name: iface.XMLName}); err != nil {
			return nil, fmt.Errorf("failed to encode interface %s: %w", iface.XMLName.Local, err)
		}
	}
	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush encoder for inner interfaces: %w", err)
	}
	return buf.Bytes(), nil
}

// BuildGetInterfacesResponse constructs the NETCONF rpc-reply for a get-interfaces request
func BuildGetInterfacesResponse(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.General.AllInterfaces", // This is the Miyagi method
			"arg": nil,
		},
		ID: 2, // Static ID for this Miyagi request
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_IF_HANDLER: Error calling Miyagi for Get.General.AllInterfaces: %v", err)
		// For custom XML, error handling might also need to be custom or simplified
		return []byte(fmt.Sprintf("Error calling Miyagi: %v", err)) // Basic error for now
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error retrieving interfaces: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_IF_HANDLER: Miyagi returned error for Get.General.AllInterfaces: %s", errMsg)
		return []byte(fmt.Sprintf("Miyagi error: %s", errMsg)) // Basic error
	}

	// miyagiResp.Result is json.RawMessage containing the map of interfaces
	var miyagiInterfaceMap map[string]MiyagiInterfaceDetail
	if err := json.Unmarshal(miyagiResp.Result, &miyagiInterfaceMap); err != nil {
		log.Printf("NETCONF_IF_HANDLER: Error unmarshalling Miyagi interface data: %v. Raw: %s", err, string(miyagiResp.Result))
		return []byte("Failed to parse interface data from device") // Basic error
	}

	// Get keys (interface names) for sorting
	var interfaceNames []string
	for name := range miyagiInterfaceMap {
		interfaceNames = append(interfaceNames, name)
	}
	// Sort interface names based on custom category and then naturally
	sort.Slice(interfaceNames, func(i, j int) bool {
		s1 := interfaceNames[i]
		s2 := interfaceNames[j]

		cat1 := getInterfaceCategory(s1)
		cat2 := getInterfaceCategory(s2)

		if cat1 != cat2 {
			return cat1 < cat2
		}
		// If in the same category, use natural sort
		return naturalSortLess(s1, s2)
	})

	var xmlInterfaces []XmlInterfaceElement
	emptyStr := "" // Reusable empty string for pointers

	for _, name := range interfaceNames {
		details := miyagiInterfaceMap[name]
		xmlEntry := XmlInterfaceElement{
			XMLName:       xml.Name{Local: name}, // Dynamic tag name
			IfDescription: details.IfDescription,
			IfPhysAddress: details.IfPhysAddress,
		}

		// Handle *int fields, converting to string or ""
		xmlEntry.IfIndex = intPtrToString(details.IfIndex)
		xmlEntry.IfType = intPtrToString(details.IfType)
		xmlEntry.IfSpeed = intPtrToString(details.IfSpeed)
		xmlEntry.IfMtu = intPtrToString(details.IfMtu)
		xmlEntry.NativeVlan = intPtrToString(details.NativeVlan)

		// Handle *int64 fields, converting to string or ""
		xmlEntry.IfInOctets = int64PtrToString(details.IfInOctets)
		xmlEntry.IfOutOctets = int64PtrToString(details.IfOutOctets)

		if details.IfAdminStatus != nil {
			xmlEntry.IfAdminStatus = &XmlStatusDescription{
				Value:       details.IfAdminStatus.Value,
				Description: details.IfAdminStatus.Description,
			}
		} else {
			xmlEntry.IfAdminStatus = &XmlStatusDescription{} // For empty <ifAdminStatus></ifAdminStatus>
		}

		if details.IfOperStatus != nil {
			xmlEntry.IfOperStatus = &XmlStatusDescription{
				Value:       details.IfOperStatus.Value,
				Description: details.IfOperStatus.Description,
			}
		} else {
			xmlEntry.IfOperStatus = &XmlStatusDescription{} // For empty <ifOperStatus></ifOperStatus>
		}

		if details.IfDuplex != nil {
			xmlEntry.IfDuplex = &XmlStatusDescription{
				Value:       details.IfDuplex.Value,
				Description: details.IfDuplex.Description,
			}
		} else {
			xmlEntry.IfDuplex = &XmlStatusDescription{}
		}

		if details.PortMode != nil {
			xmlEntry.PortMode = &XmlStatusDescription{
				Value:       details.PortMode.Value,
				Description: details.PortMode.Description,
			}
		} else {
			xmlEntry.PortMode = &XmlStatusDescription{}
		}

		if details.FlowControl != nil {
			xmlEntry.FlowControl = details.FlowControl
		} else {
			xmlEntry.FlowControl = &emptyStr // For empty <flow_control></flow_control>
		}

		if details.ComboMode == nil {
			xmlEntry.ComboMode = &emptyStr
		} else {
			xmlEntry.ComboMode = details.ComboMode
		}

		if len(details.Vlans) > 0 {
			xmlEntry.Vlan = strconv.Itoa(details.Vlans[0])
		} else {
			xmlEntry.Vlan = "" // For empty <vlans></vlans>
		}

		if len(details.UntaggedVlan) > 0 {
			xmlEntry.UntaggedVlanVal = strconv.Itoa(details.UntaggedVlan[0])
		} else {
			xmlEntry.UntaggedVlanVal = "" // For empty <untagged_vlan></untagged_vlan>
		}

		// Handle TaggedVlan
		if len(details.TaggedVlan) > 0 {
			// Assuming if it has data, we take the first element.
			// If it's meant to be a comma-separated list or always empty, adjust logic.
			xmlEntry.TaggedVlan = strconv.Itoa(details.TaggedVlan[0])
		} else {
			xmlEntry.TaggedVlan = "" // For empty <tagged_vlan></tagged_vlan>
		}

		xmlInterfaces = append(xmlInterfaces, xmlEntry)
	}

	// --- Direct Construction of XML with <rpc-reply> root ---
	innerXmlBytes, err := marshalInnerInterfaces(xmlInterfaces, "", "  ")
	if err != nil {
		log.Printf("NETCONF_IF_HANDLER: Error marshalling inner interface XML: %v", err)
		// Consider a more structured error if this were a standard NETCONF reply
		return []byte(fmt.Sprintf("Error generating inner XML content: %v", err))
	}

	var fullResponse bytes.Buffer
	fullResponse.WriteString(xml.Header)
	fullResponse.WriteString("<rpc-reply>") // Manually write the desired root tag
	if len(innerXmlBytes) > 0 {
		// Add a newline and initial indent if there's content, for pretty printing
		// This assumes the innerXmlBytes are already indented.
		// For proper nesting, the indent in marshalInnerInterfaces might need adjustment
		// or we rely on the client to pretty-print.
		// For simplicity, just append.
		fullResponse.Write(innerXmlBytes)
	}
	fullResponse.WriteString("</rpc-reply>")

	return fullResponse.Bytes()
}

// Helper to convert *int to string or "" if nil
func intPtrToString(ptr *int) string {
	if ptr == nil {
		return ""
	}
	return strconv.Itoa(*ptr)
}

// Helper to convert *int64 to string or "" if nil
func int64PtrToString(ptr *int64) string {
	if ptr == nil {
		return ""
	}
	return strconv.FormatInt(*ptr, 10)
}

// getInterfaceCategory determines the sort order for an interface name.
func getInterfaceCategory(name string) int {
	if strings.HasPrefix(name, "te") {
		return 1
	}
	if strings.HasPrefix(name, "hu") {
		return 2
	}
	if strings.HasPrefix(name, "Po") {
		return 3
	}
	if name == "oob" { // Exact match for "oob"
		return 4
	}
	if strings.HasPrefix(name, "loopback") {
		return 5
	}
	// Add other categories as needed
	// e.g., if strings.HasPrefix(name, "vl") { return 6 } for vlan interfaces

	return 100 // Default for others, sorts them last
}

// marshalToXMLIF is a helper to marshal structs to XML bytes with a standard prolog.
// This function is not used by BuildGetInterfacesResponse for the custom XML format.
func marshalToXMLIF(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_IF_HANDLER: FATAL: Failed to marshal XML: %v", err)
		// This is a programming error, should not happen with valid structs
		return []byte(fmt.Sprintf(
			`<?xml version="1.0" encoding="UTF-8"?><rpc-reply xmlns="%s"><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`,
			NetconfBaseNamespaceIF, frameEnd,
		))
	}
	// Prepend XML declaration and append frame end
	return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
}

// buildErrorResponseBytesIF creates a NETCONF <rpc-error> response.
// This function is not used by BuildGetInterfacesResponse for the custom XML format.
func buildErrorResponseBytesIF(msgID, errTag, errMsg, frameEnd string) []byte {
	reply := RpcReplyIF{
		MessageID: msgID,
		Errors: []RPCErrorIF{
			{
				ErrorType:     "application",
				ErrorTag:      errTag,
				ErrorSeverity: "error",
				ErrorMessage:  errMsg,
			},
		},
	}
	return marshalToXMLIF(reply, frameEnd)
}

// naturalSortLess provides a basic natural sort for interface names like "Po1", "Po2", "Po10".
func naturalSortLess(s1, s2 string) bool {
	extractNum := func(s string) (prefix string, num int, hasNum bool) {
		i := len(s) - 1
		for ; i >= 0; i-- {
			if s[i] < '0' || s[i] > '9' {
				break
			}
		}
		i++ // start of number part

		if i == len(s) { // No number at the end
			return s, 0, false
		}

		numPart := s[i:]
		n, err := strconv.Atoi(numPart)
		if err != nil {
			return s, 0, false // Should not happen if logic is correct
		}
		return s[:i], n, true
	}

	p1, n1, ok1 := extractNum(s1)
	p2, n2, ok2 := extractNum(s2)

	if ok1 && ok2 && p1 == p2 { // If prefixes are same and both have numbers
		return n1 < n2
	}

	// Fallback to lexicographical sort if prefixes differ or one/both don't have numbers
	return s1 < s2
}

// Note: An HandleEditConfig for interfaces would be similar to vlan.go's HandleEditConfig.
// It would require defining structs for parsing <config><interfaces>...</interfaces></config>,
// then unmarshalling the XML into these structs, and finally calling appropriate Miyagi
// "Set" methods for each interface configuration change.
