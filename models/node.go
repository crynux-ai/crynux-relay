package models

type NodeStatus uint8

const (
	NodeStatusQuit = iota
	NodeStatusAvailable
	NodeStatusBusy
	NodeStatusPendingPause
	NodeStatusPendingQuit
	NodeStatusPaused
)

type Node struct {
	Address       string      `json:"address" gorm:"index"`
	Status        NodeStatus  `json:"status" gorm:"index"`
	GPUName       string      `json:"gpu_name" gorm:"index"`
	GPUVram       uint64      `json:"gpu_vram" gorm:"index"`
	QOSScore      uint64      `json:"qos_score"`
	MajorVersion  uint64      `json:"major_version"`
	MinorVersion  uint64      `json:"minor_version"`
	PatchVersion  uint64      `json:"patch_version"`
	LastModelIDs  StringArray `json:"last_model_ids"`
	LocalModelIDs StringArray `json:"local_model_ids"`
}
