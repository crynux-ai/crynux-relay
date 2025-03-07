package nodes

import "github.com/gin-gonic/gin"

type NodeJoinInput struct {
	Address   string   `json:"address" path:"address" description:"address" validate:"required"`
	GPUName   string   `json:"gpu_name" description:"gpu_name" validate:"required"`
	GPUVram   uint64   `json:"gpu_vram" description:"gpu_vram" validate:"required"`
	Version   string   `json:"version" description:"version" validate:"required"`
	PublicKey string   `json:"public_key" description:"public key" validate:"requried"`
	ModelIDs  []string `json:"model_ids" description:"node local model ids" validate:"requried"`
}

type NodeJoinInputWithSignature struct {
	NodeJoinInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func NodeJoin(_ *gin.Context, input *NodeJoinInputWithSignature) (*NodeResponse, error) {
	return nil, nil
}
