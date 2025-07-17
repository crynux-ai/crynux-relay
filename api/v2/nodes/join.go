package nodes

import (
	"crynux_relay/api/v2/response"
	"crynux_relay/api/v2/validate"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"
	"crynux_relay/utils"
	"errors"
	"math/big"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type NodeJoinInput struct {
	Address  string        `json:"address" path:"address" description:"address" validate:"required"`
	GPUName  string        `json:"gpu_name" description:"gpu_name" validate:"required"`
	GPUVram  uint64        `json:"gpu_vram" description:"gpu_vram" validate:"required"`
	Version  string        `json:"version" description:"version" validate:"required"`
	ModelIDs []string      `json:"model_ids" description:"node local model ids" validate:"required"`
	Staking  models.BigInt `json:"staking" description:"staking amount" validate:"required"`
}

type NodeJoinInputWithSignature struct {
	NodeJoinInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func NodeJoin(c *gin.Context, in *NodeJoinInputWithSignature) (*response.Response, error) {
	match, address, err := validate.ValidateSignature(in.NodeJoinInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	if address != in.Address {
		validationErr := response.NewValidationErrorResponse("address", "Signer not allowed")
		return nil, validationErr
	}

	versions := strings.Split(in.Version, ".")
	if len(versions) != 3 {
		return nil, response.NewValidationErrorResponse("version", "Invalid node version")
	}
	nodeVersions := make([]uint64, 3)
	for i := 0; i < 3; i++ {
		if v, err := strconv.ParseUint(versions[i], 10, 64); err != nil {
			return nil, response.NewValidationErrorResponse("task_version", "Invalid task version")
		} else {
			nodeVersions[i] = v
		}
	}

	node, err := models.GetNodeByAddress(c.Request.Context(), config.GetDB(), in.Address)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		node = &models.Node{
			Address:      in.Address,
			GPUName:      in.GPUName,
			GPUVram:      in.GPUVram,
			QOSScore:     0,
			MajorVersion: nodeVersions[0],
			MinorVersion: nodeVersions[1],
			PatchVersion: nodeVersions[2],
			Status:       models.NodeStatusQuit,
		}
	} else if err != nil {
		return nil, response.NewExceptionResponse(err)
	} else {
		node.GPUName = in.GPUName
		node.GPUVram = in.GPUVram
		node.MajorVersion = nodeVersions[0]
		node.MinorVersion = nodeVersions[1]
		node.PatchVersion = nodeVersions[2]
	}
	if node.Status != models.NodeStatusQuit {
		return nil, response.NewValidationErrorResponse("address", "Node already joined")
	}

	stakeAmount := &in.Staking.Int
	if stakeAmount.Sign() == 0 {
		appConfig := config.GetConfig()
		stakeAmount = utils.EtherToWei(big.NewInt(int64(appConfig.Task.StakeAmount)))
	}

	balance, err := service.GetBalance(c.Request.Context(), config.GetDB(), in.Address)
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if balance.Cmp(stakeAmount) < 0 {
		return nil, response.NewValidationErrorResponse("balance", "Insufficient balance")
	}

	node.StakeAmount = models.BigInt{Int: *stakeAmount}

	if err := service.SetNodeStatusJoin(c.Request.Context(), config.GetDB(), node, in.ModelIDs); err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	return &response.Response{}, nil
}
