package email

import (
	"fmt"
	"time"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	"github.com/rs/zerolog/log"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func (e *sendgridEmailService) SendEmail(content string) {
	if e.client == nil {
		log.Warn().Msg("sending email failed: invalid config")
		return
	}

	if e.nextEmailSendTime.After(time.Now()) {
		log.Warn().Msgf("sending email failed: waiting for the next email send interval")
		return
	}

	from := mail.NewEmail("CudosOnDemandMintingService", e.sender)
	subject := "CudosOnDemandMintingService - Error"
	to := mail.NewEmail("CudosServiceEmail", e.recipient)
	message := mail.NewSingleEmail(from, subject, to, content, "")

	res, err := e.client.Send(message)
	if err != nil {
		log.Warn().Err(fmt.Errorf("sending email failed: %s", err)).Send()
		return
	}

	if res.StatusCode >= 400 {
		log.Warn().Err(fmt.Errorf("sending email failed: %s", res.Body)).Send()
		return
	}

	e.nextEmailSendTime = time.Now().Add(e.emailSendInterval)
	log.Info().Msg("sending email succeeded")
}

func NewSendgridEmailService(cfg config.Config) *sendgridEmailService {
	if !cfg.HasValidEmailConfig() {
		log.Warn().Msg("creating email service failed: invalid config")
		return &sendgridEmailService{}
	}

	return &sendgridEmailService{
		client:            sendgrid.NewSendClient(cfg.SendgridApiKey),
		sender:            cfg.EmailFrom,
		recipient:         cfg.ServiceEmail,
		emailSendInterval: cfg.EmailSendInterval,
		nextEmailSendTime: time.Now(),
	}
}

type sendgridEmailService struct {
	client            *sendgrid.Client
	sender            string
	recipient         string
	emailSendInterval time.Duration
	nextEmailSendTime time.Time
}
