package v1

import (
	"crynux_relay/api/v1/inference_tasks"
	"crynux_relay/api/v1/network"
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/time"
	"crynux_relay/api/v1/worker"

	"github.com/loopfz/gadgeto/tonic"
	"github.com/wI2L/fizz"
)

func InitRoutes(r *fizz.Fizz) {

	v1g := r.Group("v1", "ApiV1", "API version 1")

	v1g.GET("now", []fizz.OperationOption{
		fizz.Summary("Get current unix timestamp of server"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(time.GetNow, 200))

	tasksGroup := v1g.Group("inference_tasks", "Inference tasks", "Inference tasks related APIs")

	tasksGroup.POST("", []fizz.OperationOption{
		fizz.Summary("Create an task"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
		fizz.Response("500", "exception", response.ExceptionResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.CreateTask, 200))

	tasksGroup.GET("/:task_id", []fizz.OperationOption{
		fizz.Summary("Get a task by task id"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.GetTaskById, 200))

	tasksGroup.POST("/:task_id/results", []fizz.OperationOption{
		fizz.Summary("Upload task result"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
		fizz.Response("500", "exception", response.ExceptionResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.UploadResult, 200))

	tasksGroup.GET("/:task_id/results/:image_num", []fizz.OperationOption{
		fizz.Summary("Get the result of the task by node address"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.GetResult, 200))
	tasksGroup.GET("/:task_id/results/checkpoint", []fizz.OperationOption{
		fizz.Summary("Get the result checkpoint of the task by node address"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.GetResultCheckpoint, 200))

	tasksGroup.POST("/:task_id/checkpoint", []fizz.OperationOption{
		fizz.Summary("Upload the input checkpoint of the task"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.UploadCheckpoint, 200))
	tasksGroup.GET("/:task_id/checkpoint", []fizz.OperationOption{
		fizz.Summary("Get the input checkpoint of the task"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.GetCheckpoint, 200))


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

	networkGroup.GET("", []fizz.OperationOption{
		fizz.Summary("Get total TFLOPS of the network"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(network.GetNetworkTFLOPS, 200))

	workerGroup := v1g.Group("worker", "worker", "Worker count related APIs")

	workerGroup.POST("/:version", []fizz.OperationOption{
		fizz.Summary("Called when a worker is up"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(worker.WorkerJoin, 200))

	workerGroup.DELETE("/:version", []fizz.OperationOption{
		fizz.Summary("Called when a worker is down"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(worker.WorkerQuit, 200))

	workerGroup.GET("/:version/count", []fizz.OperationOption{
		fizz.Summary("Get worker count of specified version"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(worker.GetWorkerCount, 200))
}
