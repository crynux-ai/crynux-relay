package v2

import (
	"crynux_relay/api/v2/incentive"
	"crynux_relay/api/v2/network"
	"crynux_relay/api/v2/nodes"
	"crynux_relay/api/v2/response"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

func InitRoutes(r *fizz.Fizz) {

	v2g := r.Group("v2", "ApiV2", "API version 2")

	incentiveGroup := v2g.Group("incentive", "incentive", "incentive statistics related APIs")

	incentiveGroup.GET("/nodes", []fizz.OperationOption{
		fizz.ID("incentive_nodes_v2"),
		fizz.Summary("Get nodes with top K incentive"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(incentive.GetNodeIncentive, 200))

	networkGroup := v2g.Group("network", "network", "Network stats related APIs")

	networkGroup.GET("/nodes/data", []fizz.OperationOption{
		fizz.ID("network_nodes_data_v2"),
		fizz.Summary("Get the info of all the nodes in the network"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(network.GetAllNodeData, 200))

	nodeGroup := v2g.Group("node", "node", "Node APIs")

	nodeGroup.GET("/:address", []fizz.OperationOption{
		fizz.ID("node_get_v2"),
		fizz.Summary("Get node info"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(nodes.GetNode, 200))
}
