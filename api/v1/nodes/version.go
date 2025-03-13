package nodes

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/validate"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UpdateVersionInput struct {
	Address string `json:"address" path:"address" description:"address" validate:"required"`
	Version string `json:"version" description:"new node version" validate:"required"`
}

type UpdateVersionInputWithSignature struct {
	UpdateVersionInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func UpdateNodeVersion(c *gin.Context, in *UpdateVersionInputWithSignature) (*response.Response, error) {
	match, address, err := validate.ValidateSignature(in.UpdateVersionInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	if in.Address != address {
		return nil, response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	node, err := models.GetNodeByAddress(c.Request.Context(), config.GetDB(), in.Address)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("address", "Node not found")
			return nil, validationErr
		}
		return nil, response.NewExceptionResponse(err)
	}

	if node.Status == models.NodeStatusQuit {
		return nil, response.NewValidationErrorResponse("address", "Illegal node status")
	}

	versions := strings.Split(in.Version, ".")
	if len(versions) != 3 {
		return nil, response.NewValidationErrorResponse("version", "Invalid node version")
	}
	nodeVersions := make([]uint64, 3)
	for i := 0; i < 3; i++ {
		if v, err := strconv.ParseUint(versions[i], 10, 64); err != nil {
			return nil, response.NewValidationErrorResponse("version", "Invalid node version")
		} else {
			nodeVersions[i] = v
		}
	}

	if err := node.Update(c.Request.Context(), config.GetDB(), &models.Node{MajorVersion: nodeVersions[0], MinorVersion: nodeVersions[1], PatchVersion: nodeVersions[2]}); err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	return &response.Response{}, nil
}
