package v1

import (
	"crynux_relay/api/v1/incentive"
	"crynux_relay/api/v1/inference_tasks"
	"crynux_relay/api/v1/network"
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/stats"
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

	tasksGroup.POST("/:task_id_commitment", []fizz.OperationOption{
		fizz.Summary("Create an task"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
		fizz.Response("500", "exception", response.ExceptionResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.CreateTask, 200))

	tasksGroup.GET("/:task_id_commitment", []fizz.OperationOption{
		fizz.Summary("Get a task by task id"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.GetTaskById, 200))

	tasksGroup.POST("/:task_id_commitment/results", []fizz.OperationOption{
		fizz.Summary("Upload task result"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
		fizz.Response("500", "exception", response.ExceptionResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.UploadResult, 200))

	tasksGroup.GET("/:task_id_commitment/results/:index", []fizz.OperationOption{
		fizz.Summary("Get the result of the task by node address"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.GetResult, 200))
	tasksGroup.GET("/:task_id_commitment/results/checkpoint", []fizz.OperationOption{
		fizz.Summary("Get the result checkpoint of the task by node address"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(inference_tasks.GetResultCheckpoint, 200))

	tasksGroup.GET("/:task_id_commitment/checkpoint", []fizz.OperationOption{
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

	statsGroup := v1g.Group("stats", "stats", "task statistics related APIs")

	statsGroup.GET("/line_chart/task_count", []fizz.OperationOption{
		fizz.Summary("Get line chart data of task count"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(stats.GetTaskCountLineChart, 200))

	statsGroup.GET("/line_chart/task_success_rate", []fizz.OperationOption{
		fizz.Summary("Get line chart data of task success rate"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(stats.GetTaskSuccessRateLineChart, 200))

	statsGroup.GET("/histogram/task_execution_time", []fizz.OperationOption{
		fizz.Summary("Get histogram data of task execution time"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(stats.GetTaskExecutionTimeHistogram, 200))
	statsGroup.GET("/histogram/task_upload_result_time", []fizz.OperationOption{
		fizz.Summary("Get histogram data of task upload result time"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(stats.GetTaskUploadResultTimeHistogram, 200))
	statsGroup.GET("/histogram/task_waiting_time", []fizz.OperationOption{
		fizz.Summary("Get histogram data of task waiting time"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(stats.GetTaskWaitingTimeHistogram, 200))

	statsGroup.GET("/line_chart/incentive", []fizz.OperationOption{
		fizz.Summary("Get line chart data of incentives"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(stats.GetIncentiveLineChart, 200))
	statsGroup.GET("/histogram/task_fee", []fizz.OperationOption{
		fizz.Summary("Get histogram data of task fee in the path hour"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(stats.GetTaskFeeHistogram, 200))

	statsGroup.GET("/node_events", []fizz.OperationOption{
		fizz.Summary("Get node event logs in the recent hour"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(stats.GetNodeEventLogs, 200))

	incentiveGroup := v1g.Group("incentive", "incentive", "incentive statistics related APIs")

	incentiveGroup.GET("/total", []fizz.OperationOption{
		fizz.Summary("Get today's total incentive"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(incentive.GetTotalIncentive, 200))

	incentiveGroup.GET("/nodes", []fizz.OperationOption{
		fizz.Summary("Get nodes with top K incentive"),
		fizz.Response("400", "validation errors", response.ValidationErrorResponse{}, nil, nil),
	}, tonic.Handler(incentive.GetNodeIncentive, 200))

}
