package nodes

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/validate"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type PauseInput struct {
	Address string `path:"address" json:"address" description:"address" validate:"required"`
}

type PauseInputWithSignature struct {
	PauseInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func NodePause(c *gin.Context, in *PauseInputWithSignature) (*response.Response, error) {
	match, address, err := validate.ValidateSignature(in.PauseInput, in.Timestamp, in.Signature)

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

	for range 3 {
		var status models.NodeStatus
		switch node.Status {
		case models.NodeStatusAvailable:
			status = models.NodeStatusPaused
		case models.NodeStatusBusy:
			status = models.NodeStatusPendingPause
		default:
			return nil, response.NewValidationErrorResponse("address", "Illegal node status")
		}

		err = node.Update(c.Request.Context(), config.GetDB(), map[string]interface{}{"status": status})
		if err == nil {
			break
		} else if errors.Is(err, models.ErrNodeStatusChanged) {
			if err := node.SyncStatus(c.Request.Context(), config.GetDB()); err != nil {
				return nil, response.NewExceptionResponse(err)
			}
		} else {
			return nil, response.NewExceptionResponse(err)
		}
	}
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	return &response.Response{}, nil
}
