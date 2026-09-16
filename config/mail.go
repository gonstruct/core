package config

type Mailer string

const (
	MailerSMTP      Mailer = "smtp"
	MailerAmazonSES Mailer = "amazon_ses"
)

func (Mailer) Values() []Mailer {
	return []Mailer{
		MailerSMTP,
		MailerAmazonSES,
	}
}

func (m Mailer) String() string {
	return string(m)
}

type MailFrom struct {
	Name    string
	Address string

	ReplyToName    string
	ReplyToAddress string

	Bcc string
}

type Mail struct {
	Mailer   Mailer
	Region   string
	Host     string
	Port     int
	Username string
	Password string
	From     MailFrom
}
