package nodes

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/models"
)

type Node struct {
	Address       string            `json:"address" gorm:"index"`
	Status        models.NodeStatus `json:"status" gorm:"index"`
	GPUName       string            `json:"gpu_name" gorm:"index"`
	GPUVram       uint64            `json:"gpu_vram" gorm:"index"`
	QOSScore      float64           `json:"qos_score"`
	StakingScore  float64           `json:"staking_score"`
	ProbWeight    float64           `json:"prob_weight"`
	Version       string            `json:"version"`
	InUseModelIDs []string          `json:"in_use_model_ids"`
	ModelIDs      []string          `json:"model_ids"`
}

type NodeResponse struct {
	response.Response
	Data *Node `json:"data"`
}
