package masking

// DefaultKeys are lowercase substrings matched against JSON keys. Matching is by
// substring, so "email" also covers "email_address" and "user_email".
var DefaultKeys = []string{
	// Authentication and secrets
	"password",
	"passwd",
	"secret",
	"token",
	"access_token",
	"refresh_token",
	"id_token",
	"api_key",
	"apikey",
	"authorization",
	"private_key",
	"secret_key",
	"client_secret",
	"webhook_secret",

	// Personally identifiable information
	"email",
	"phone",
	"mobile",
	"address",
	"date_of_birth",
	"dob",
	"birth",

	// Financial
	"credit_card",
	"card_number",
	"cvv",
	"cvc",
	"ssn",
	"iban",
	"account_number",
}
