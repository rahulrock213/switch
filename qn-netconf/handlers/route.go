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

const RoutingNamespace = "urn:example:params:xml:ns:yang:routing"
const NetconfBaseNamespaceRoute = "urn:ietf:params:xml:ns:netconf:base:1.0"

// --- Common NETCONF XML Data Structures (for Route handler) ---

type RpcReplyRoute struct {
	XMLName       xml.Name                `xml:"rpc-reply"`
	RoutingConfig *RoutingDataGetResponse `xml:"routing,omitempty"` // For GET response
	Ok            *OkRoute                `xml:"ok,omitempty"`      // For edit-config response
	Errors        []RPCErrorRoute         `xml:"rpc-error,omitempty"`
}

type OkRoute struct {
	XMLName xml.Name `xml:"ok"`
}

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
	RoutingData *RoutingData `xml:"routing"` // Matches <routing> by local name
}

// RoutingData corresponds to the <routing> container
type RoutingData struct {
	// XMLName made namespace-agnostic for flexible unmarshalling in edit-config.
	// Namespace will be explicitly set when marshalling for GET response.
	XMLName      xml.Name          `xml:"routing"` // For edit-config unmarshalling
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
	XMLName   xml.Name `xml:"route"`                    // For edit-config
	Operation string   `xml:"operation,attr,omitempty"` // For 'create', 'delete', 'merge', etc.
	Prefix    string   `xml:"prefix"`
	Mask      string   `xml:"mask"`
	NextHop   string   `xml:"next-hop,omitempty"` // Optional for delete
}

// --- Structures for GET response ---

// RoutingDataGetResponse is for the <routing> container in a GET response.
type RoutingDataGetResponse struct {
	XMLName      xml.Name                     `xml:"yang:route routing"` // Namespace for GET response
	StaticRoutes *StaticRoutesDataGetResponse `xml:"static-routes"`
}

// StaticRoutesDataGetResponse corresponds to <static-routes> in a GET response.
type StaticRoutesDataGetResponse struct {
	XMLName xml.Name               `xml:"static-routes"`
	Routes  []RouteDataGetResponse `xml:"route"`
}

// RouteDataGetResponse corresponds to a single <route> in a GET response.
type RouteDataGetResponse struct {
	XMLName xml.Name `xml:"route"` // No operation attribute for GET
	Prefix  string   `xml:"prefix"`
	Mask    string   `xml:"mask"`
	NextHop string   `xml:"next-hop"`
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

	reply := RpcReplyRoute{
		// MessageID is no longer part of RpcReplyRoute
		Ok: &OkRoute{},
	}
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
		// MessageID is no longer part of RpcReplyRoute
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

// HandleRouteGetConfig handles <get> or <get-config> for static routes
func HandleRouteGetConfig(miyagiSocketPath, msgID, frameEnd string) []byte {
	miyagiReq := miyagi.MiyagiRequest{
		Method: "call",
		Params: map[string]interface{}{
			"uid": "Agent.Switch.Get.IPRouting.Table", // Assumed Miyagi UID
			"arg": nil,
		},
		ID: 9, // Static ID for this Miyagi request
	}

	miyagiResp, err := miyagi.SendRequest(miyagiSocketPath, miyagiReq)
	if err != nil {
		log.Printf("NETCONF_ROUTE_HANDLER: Error calling Miyagi for Get.IPRouting.Table: %v", err)
		return buildErrorResponseBytesRoute(msgID, "application", "operation-failed", "Error communicating with device agent for routes", frameEnd)
	}

	if miyagiResp.Error != nil {
		errMsg := fmt.Sprintf("Device error retrieving routes: %s (code: %d)", miyagiResp.Error.Message, miyagiResp.Error.Code)
		log.Printf("NETCONF_ROUTE_HANDLER: Miyagi returned error for Get.IPRouting.Table: %s", errMsg)
		return buildErrorResponseBytesRoute(msgID, "application", "operation-failed", errMsg, frameEnd)
	}

	// Assuming Miyagi returns a JSON array of route objects
	var miyagiRoutes []RouteDataGetResponse // Use the GET response struct directly
	if err := json.Unmarshal(miyagiResp.Result, &miyagiRoutes); err != nil {
		log.Printf("NETCONF_ROUTE_HANDLER: Error unmarshalling Miyagi route table: %v. Raw: %s", err, string(miyagiResp.Result))
		return buildErrorResponseBytesRoute(msgID, "application", "operation-failed", "Failed to parse route data from device", frameEnd)
	}

	// Prepare the data for XML marshalling
	var routesForXML []RouteDataGetResponse
	for _, r := range miyagiRoutes {
		routesForXML = append(routesForXML, RouteDataGetResponse{
			// XMLName will be set by the struct tag "route"
			Prefix:  r.Prefix,
			Mask:    r.Mask,
			NextHop: r.NextHop,
		})
	}

	routingConfig := RoutingDataGetResponse{
		// XMLName includes "yang:route routing"
		StaticRoutes: &StaticRoutesDataGetResponse{
			Routes: routesForXML,
		},
	}

	reply := RpcReplyRoute{
		// MessageID is no longer part of RpcReplyRoute
		RoutingConfig: &routingConfig,
	}

	return marshalToXMLRoute(reply, frameEnd)
}
