package api

import (
	"github.com/matcornic/hermes/v2"
)

func generateEmail(userEmail string, resetToken string) string {

	// Setup global Hermes options
	h := hermes.Hermes{
		Product: hermes.Product{
			Name: "Petkeep",
			Link: "https://www.petkeep.com",
		},
	}
	// Create the hermes.Email
	email := hermes.Email{
		Body: hermes.Body{
			Name: userEmail,
			Intros: []string{
				"A password reset has been requested within Petkeep.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "To reset your password, click here:",
					Button: hermes.Button{
						Color: "#4A9FFA", // Optional action button color
						Text:  "Reset Password",
						Link:  "https://hermes-example.com/confirm?token=d9729feb74992cc3482b350163a1a010",
					},
				},
			},
			Outros: []string{
				"Need help, or have questions? Just reply to this email, we'd love to help.",
			},
		},
	}
	// Generate an HTML email with the provided contents (for modern clients)
	emailBody, err := h.GenerateHTML(email)
	if err != nil {
		panic(err) // Tip: Handle error with something else than a panic ;)
	}
	return emailBody
}
