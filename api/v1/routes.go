package v1

import (
	"crynux_relay/api/v1/inference_tasks"
	"crynux_relay/api/v1/network"
	"crynux_relay/api/v1/response"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

func InitRoutes(r *fizz.Fizz) {

	v1g := r.Group("v1", "ApiV1", "API version 1")

	tasksGroup := v1g.Group("inference_tasks", "Inference tasks", "Inference tasks related APIs")

	tasksGroup.POST("", []fizz.OperationOption{
		fizz.Summary("Create an inference task"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
		fizz.Response("500", "exception", response.ExceptionResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.CreateTask, 200))

	tasksGroup.GET("/:task_id", []fizz.OperationOption{
		fizz.Summary("Get an inference task by task id"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.GetTaskById, 200))

	tasksGroup.POST("/:task_id/results", []fizz.OperationOption{
		fizz.Summary("Upload inference task result"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
		fizz.Response("500", "exception", response.ExceptionResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.UploadResult, 200))

	tasksGroup.GET("/:task_id/results/:image_num", []fizz.OperationOption{
		fizz.Summary("Get the result of the inference task by node address"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.GetResult, 200))

	networkGroup := v1g.Group("network", "network", "Network stats related APIs")

	networkGroup.GET("/nodes/data", []fizz.OperationOption{
		fizz.Summary("Get the info of all the nodes in the network"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(network.GetAllNodeData, 200))

	networkGroup.GET("/nodes/number", []fizz.OperationOption{
		fizz.Summary("Get total nodes number in the network"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(network.GetAllNodeNumber, 200))

	networkGroup.GET("/tasks/number", []fizz.OperationOption{
		fizz.Summary("Get total task number in the network"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(network.GetAllTaskNumber, 200))


}
