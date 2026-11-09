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

var _ model.ClientCtx = Client{}
var _ model.DriverCtx = Driver{}

// NewClientCtx adds context support without changing the legacy driver contract.
func (d Driver) NewClientCtx(params model.DriverClientParam, options model.ClientOptions) (model.ClientCtx, error) {
	if options.Timeout < 0 {
		return nil, errors.ErrDriverCredentialInvalid.WithDriverName(Name)
	}
	if params.Credential[SmsAccount] == "" {
		return nil, errors.ErrDriverCredentialInvalid.WithDriverName(Name)
	}

	legacy, err := d.NewClient(params)
	if err != nil {
		return nil, err
	}
	c := legacy.(Client)
	if options.Transport != nil || options.Timeout > 0 {
		h := *c.Client.Client.Client
		if options.Transport != nil {
			h.Transport = options.Transport
		}
		if options.Timeout > 0 {
			h.Timeout = options.Timeout
		}
		c.Client.Client.Client = &h
	}
	return c, nil
}
