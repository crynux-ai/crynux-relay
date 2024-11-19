package stats

type TaskTypeString string

const (
	ImageTaskType TaskTypeString = "Image"
	TextTaskType  TaskTypeString = "Text"
	AllTaskType   TaskTypeString = "All"
)

type TimeUnit string

const (
	UnitHour  TimeUnit = "Hour"
	UnitDay   TimeUnit = "Day"
	UnitWeek  TimeUnit = "Week"
	UnitMonth TimeUnit = "Month"
)
