package handlers

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"strings"

	"qn-netconf/miyagi"
)

const RoutingNamespace = "urn:example:params:xml:ns:yang:routing"
const NetconfBaseNamespaceRoute = "urn:ietf:params:xml:ns:netconf:base:1.0"

// --- Common NETCONF XML Data Structures (for Route handler) ---

type RpcReplyRoute struct {
	XMLName   xml.Name        `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
	MessageID string          `xml:"message-id,attr"`
	Ok        *OkRoute        `xml:"ok,omitempty"`
	Errors    []RPCErrorRoute `xml:"rpc-error,omitempty"`
	// Data      *DataRoute    `xml:"data,omitempty"` // Not used for edit-config, but would be for get-config
}

type OkRoute struct {
	XMLName xml.Name `xml:"ok"`
}

// type DataRoute struct { // Example if we were to implement get-config for routes
// 	XMLName     xml.Name     `xml:"data"`
// 	RoutingData *RoutingData `xml:"routing,omitempty"`
// }

type RPCErrorRoute struct {
	XMLName       xml.Name `xml:"rpc-error"`
	ErrorType     string   `xml:"error-type"`
	ErrorTag      string   `xml:"error-tag"`
	ErrorSeverity string   `xml:"error-severity"`
	ErrorMessage  string   `xml:"error-message"`
}

// --- Routing Specific XML Data Structures for <edit-config> ---

// EditConfigRoutingPayload is the top-level structure for <config>
type EditConfigRoutingPayload struct {
	XMLName     xml.Name     `xml:"config"`
	RoutingData *RoutingData `xml:"routing"`
}

// RoutingData corresponds to the <routing> container
type RoutingData struct {
	XMLName      xml.Name          `xml:"routing"`
	Xmlns        string            `xml:"xmlns,attr,omitempty"`
	StaticRoutes *StaticRoutesData `xml:"static-routes"`
}

// StaticRoutesData corresponds to the <static-routes> container
type StaticRoutesData struct {
	XMLName xml.Name    `xml:"static-routes"`
	Routes  []RouteData `xml:"route"` // Can have multiple route operations in one edit-config
}

// RouteData corresponds to a single <route> entry
type RouteData struct {
	XMLName   xml.Name `xml:"route"`
	Operation string   `xml:"operation,attr,omitempty"` // For 'create', 'delete', 'merge', etc.
	Prefix    string   `xml:"prefix"`
	Mask      string   `xml:"mask"`
	NextHop   string   `xml:"next-hop,omitempty"` // Optional for delete
}

// --- Handler Function ---

// HandleRouteEditConfig handles <edit-config> for static routes
func HandleRouteEditConfig(miyagiSocketPath string, request []byte, msgID, frameEnd string) []byte {
	var editReq EditConfigRoutingPayload
	configStartIndex := bytes.Index(request, []byte("<config>"))
	configEndIndex := bytes.LastIndex(request, []byte("</config>"))

	if configStartIndex == -1 || configEndIndex == -1 || configStartIndex >= configEndIndex {
		log.Printf("NETCONF_ROUTE_HANDLER: Malformed <edit-config> request, <config> tag not found or invalid: %s", string(request))
		return buildErrorResponseBytesRoute(msgID, "protocol", "malformed-message", "Malformed <edit-config> request", frameEnd)
	}
	configPayload := request[configStartIndex : configEndIndex+len("</config>")]

	if err := xml.Unmarshal(configPayload, &editReq); err != nil {
		log.Printf("NETCONF_ROUTE_HANDLER: Error unmarshalling routing <edit-config> payload: %v. Payload: %s", err, string(configPayload))
		return buildErrorResponseBytesRoute(msgID, "protocol", "malformed-message", "Invalid routing configuration format", frameEnd)
	}

	if editReq.RoutingData == nil || editReq.RoutingData.StaticRoutes == nil || len(editReq.RoutingData.StaticRoutes.Routes) == 0 {
		log.Printf("NETCONF_ROUTE_HANDLER: Malformed routing <edit-config>, <routing><static-routes><route> structure missing or empty.")
		return buildErrorResponseBytesRoute(msgID, "protocol", "missing-element", "<routing><static-routes><route> is required", frameEnd)
	}

	// Process each route operation
	// For simplicity, this example processes one route at a time.
	// A more robust implementation might batch or handle dependencies.
	for _, route := range editReq.RoutingData.StaticRoutes.Routes {
		var miyagiUID string
		var miyagiArgs map[string]interface{}
		miyagiReqID := 7 // Default, can be incremented or made unique per call

		switch route.Operation {
		case "create", "merge", "replace": // Treat merge/replace as create for this simple example
			if route.Prefix == "" || route.Mask == "" || route.NextHop == "" {
				return buildErrorResponseBytesRoute(msgID, "protocol", "missing-attribute", "For create operation, prefix, mask, and next-hop are required.", frameEnd)
			}
			miyagiUID = "Agent.Switch.Set.IPRouting.Create"
			miyagiArgs = map[string]interface{}{
				"prefix":   route.Prefix,
				"mask":     route.Mask,
				"next_hop": route.NextHop,
			}
			miyagiReqID = 7
		case "delete":
			if route.Prefix == "" || route.Mask == "" {
				return buildErrorResponseBytesRoute(msgID, "protocol", "missing-attribute", "For delete operation, prefix and mask are required.", frameEnd)
			}
			miyagiUID = "Agent.Switch.Set.IPRouting.Remove"
			miyagiArgs = map[string]interface{}{
				"prefix": route.Prefix,
				"mask":   route.Mask,
			}
			miyagiReqID = 8
		default:
			log.Printf("NETCONF_ROUTE_HANDLER: Unsupported route operation '%s' for prefix %s", route.Operation, route.Prefix)
			return buildErrorResponseBytesRoute(msgID, "protocol", "bad-attribute", fmt.Sprintf("Unsupported operation: %s", route.Operation), frameEnd)
		}

		miyagiReq := miyagi.MiyagiRequest{
			Method: "call",
			Params: map[string]interface{}{"uid": miyagiUID, "arg": miyagiArgs},
			ID:     miyagiReqID,
		}

		miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
		if err != nil {
			log.Printf("NETCONF_ROUTE_HANDLER: Error calling Miyagi for %s (prefix %s): %v", miyagiUID, route.Prefix, err)
			return buildErrorResponseBytesRoute(msgID, "application", "operation-failed", fmt.Sprintf("Error communicating with device agent for route %s", route.Prefix), frameEnd)
		}

		if miyagiResp.Error != nil {
			errMsg := fmt.Sprintf("Device error processing route %s (operation %s): %s (code: %d)", route.Prefix, route.Operation, miyagiResp.Error.Message, miyagiResp.Error.Code)
			log.Printf("NETCONF_ROUTE_HANDLER: Miyagi returned error for %s: %s", miyagiUID, errMsg)
			return buildErrorResponseBytesRoute(msgID, "application", "operation-failed", errMsg, frameEnd)
		}
		// If multiple routes, continue. If any fails, we return error.
		// For simplicity, we stop at first error.
	}

	reply := RpcReplyRoute{MessageID: msgID, Ok: &OkRoute{}}
	return marshalToXMLRoute(reply, frameEnd)
}

// --- Helper Functions (specific to Route handler) ---

func marshalToXMLRoute(data interface{}, frameEnd string) []byte {
	xmlBytes, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("NETCONF_ROUTE_HANDLER: FATAL: Failed to marshal XML: %v", err)
		return []byte(fmt.Sprintf(
			`<?xml version="1.0" encoding="UTF-8"?><rpc-reply xmlns="%s"><rpc-error><error-type>application</error-type><error-tag>internal-error</error-tag><error-severity>error</error-severity><error-message>Internal server error during XML generation</error-message></rpc-error></rpc-reply>%s`,
			NetconfBaseNamespaceRoute, frameEnd,
		))
	}
	return append([]byte(xml.Header), append(xmlBytes, []byte(frameEnd)...)...)
}

func buildErrorResponseBytesRoute(msgID, errType, errTag, errMsg, frameEnd string) []byte {
	escapedErrMsg := strings.ReplaceAll(errMsg, "<", "&lt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, ">", "&gt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, "&", "&amp;")

	reply := RpcReplyRoute{
		MessageID: msgID,
		Errors: []RPCErrorRoute{
			{
				ErrorType:     errType,
				ErrorTag:      errTag,
				ErrorSeverity: "error",
				ErrorMessage:  escapedErrMsg,
			},
		},
	}
	return marshalToXMLRoute(reply, frameEnd)
}
