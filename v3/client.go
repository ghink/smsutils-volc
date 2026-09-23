package volc

import (
	"github.com/volcengine/volc-sdk-golang/service/sms"
	"go.gh.ink/smsutils/v3/errors"
	"go.gh.ink/smsutils/v3/model"
)

type Client struct {
	// Client is this channel's own SDK instance. The volc SDK also exports a
	// package-level DefaultInstance; configuring keys there would make every
	// channel built in one process share whichever credentials were set last.
	Client     *sms.SMS
	SmsAccount string
	// JSON
	Marshal   func(any) ([]byte, error)
	Unmarshal func([]byte, any) error
}
type Driver struct{}

func (d Driver) NewClient(params model.DriverClientParam) (model.Client, error) {
	// Check credential
	ak, sk := params.Credential[AccessKey], params.Credential[SecretKey]
	if ak == "" || sk == "" {
		return Client{}, errors.ErrDriverCredentialInvalid.WithDriverName(Name)
	}

	// Create volc client
	client := sms.NewInstance()

	// Set access key and secret key
	client.Client.SetAccessKey(ak)
	client.Client.SetSecretKey(sk)

	return Client{
		Client:     client,
		SmsAccount: params.Credential[SmsAccount],
		Marshal:    params.Marshal,
		Unmarshal:  params.Unmarshal,
	}, nil
}
