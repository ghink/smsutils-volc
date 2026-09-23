package volc

import (
	"strconv"

	"github.com/volcengine/volc-sdk-golang/service/sms"
	"go.gh.ink/smsutils/v3/errors"
	"go.gh.ink/smsutils/v3/model"
	"go.gh.ink/smsutils/v3/utils"
)

func (c Client) SendMessage(dest string, sender string, template string, vars model.Vars) error {
	// Try to parse number
	dest, _, _, _, err := utils.ProcessNumberForChinese(dest)
	if err != nil {
		return err
	}

	// Preprocess vars
	params := make(map[string]string)
	for _, v := range vars {
		params[v.Key] = v.Value
	}

	// Marshal params
	marshalledParam, err := c.Marshal(params)
	if err != nil {
		return err
	}

	// Construct request
	req := &sms.SmsRequest{
		SmsAccount:    c.SmsAccount,
		Sign:          sender,
		TemplateID:    template,
		TemplateParam: string(marshalledParam),
		PhoneNumbers:  dest,
	}

	// Send request
	result, statusCode, err := c.Client.Send(req)
	if err != nil {
		return err
	}
	if statusCode != 200 {
		return errors.ErrDriverSendFailed.
			WithDriverName(Name).
			WithDriverCode(strconv.Itoa(statusCode)).
			WithDriverRequestID(result.ResponseMetadata.RequestId).
			WithDriverResponse(result)
	}
	if result.ResponseMetadata.Error != nil {
		return errors.ErrDriverSendFailed.
			WithDriverName(Name).
			WithDriverCode(strconv.Itoa(statusCode)).
			WithDriverMessage(result.ResponseMetadata.Error.Message).
			WithDriverRequestID(result.ResponseMetadata.RequestId).
			WithDriverResponse(result)
	}

	return nil
}
