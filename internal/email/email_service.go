package email

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

var (
	lastSentEmailTimestamp = int64(0)
)

func SendEmail(cfg *config.Config, content string) {
	if !cfg.HasValidEmailSettings() {
		log.Info().Msg("Emails are not configured")
		return
	}

	if time.Now().Unix()-lastSentEmailTimestamp > 1800 {
		lastSentEmailTimestamp = time.Now().Unix()

		from := mail.NewEmail("CudosOnDemandMintingService", cfg.EmailFrom)
		subject := "CudosOnDemandMintingService - Error"
		to := mail.NewEmail("CudosServiceEmail", cfg.ServiceEmail)
		plainTextContent := content
		message := mail.NewSingleEmail(from, subject, to, plainTextContent, "")
		client := sendgrid.NewSendClient(cfg.SendgridApiKey)
		_, err := client.Send(message)
		if err != nil {
			log.Error().Err(err).Send()
		} else {
			log.Info().Msgf("Service email successfully sent")
			// fmt.Println(response.StatusCode)
			// fmt.Println(response.Body)
			// fmt.Println(response.Headers)
		}
	} else {
		log.Info().Msgf("Next error email can be send no sooner than %ds", (1800 - (time.Now().Unix() - lastSentEmailTimestamp)))
	}
}
