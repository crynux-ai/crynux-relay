package balance

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/validate"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type TransferInput struct {
	From  string   `path:"from" json:"from" description:"The from address of the transfer request"`
	To    string   `json:"to" description:"The to address of the transfer request" validate:"required"`
	Value models.BigInt `json:"value" description:"The transferred value" validate:"required"`
}

type TransferInputWithSignature struct {
	TransferInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func Transfer(c *gin.Context, in *TransferInputWithSignature) (*response.Response, error) {
	match, address, err := validate.ValidateSignature(in.TransferInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	if address != in.From {
		validationErr := response.NewValidationErrorResponse("from", "Signer not allowed")
		return nil, validationErr
	}

	commitFunc, err := service.Transfer(c.Request.Context(), config.GetDB(), in.From, in.To, &in.Value.Int)
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	commitFunc()
	return &response.Response{}, nil
}
