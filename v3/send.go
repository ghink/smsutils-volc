package volc

import (
	"context"
	"strconv"

	"github.com/volcengine/volc-sdk-golang/base"
	"github.com/volcengine/volc-sdk-golang/service/sms"
	"go.gh.ink/smsutils/v3/errors"
	"go.gh.ink/smsutils/v3/model"
	"go.gh.ink/smsutils/v3/utils"
)

// SendMessage retains the original API and delegates to the context-aware path.
func (c Client) SendMessage(dest string, sender string, template string, vars model.Vars) error {
	return c.SendMessageCtx(context.Background(), dest, sender, template, vars)
}

// SendMessageCtx binds cancellation to the actual provider HTTP request.
func (c Client) SendMessageCtx(ctx context.Context, dest, sender, template string, vars model.Vars) (err error) {
	if ctx == nil {
		return errors.ErrMessageInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	defer func() {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	}()
	if err := utils.ValidateVars(vars, true); err != nil {
		return err
	}

	// Try to parse number
	dest, _, _, _, err = utils.ProcessNumberForChinese(dest)
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
	if c.Client == nil || c.Client.Client == nil {
		return errors.ErrDriverSendFailed.WithDriverName(Name)
	}
	raw, err := c.Marshal(req)
	if err != nil {
		return err
	}
	sdk := *c.Client.Client
	service := *sdk.ServiceInfo
	service.Retry = base.RetrySettings{AutoRetry: false}
	sdk.ServiceInfo = &service
	rawResponse, statusCode, err := sdk.CtxJson(ctx, "SendSms", nil, string(raw))
	var result sms.SmsResponse
	if err == nil {
		err = c.Unmarshal(rawResponse, &result)
	}
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

	if result.Result == nil || len(result.Result.MessageID) == 0 {
		return errors.ErrDriverSendFailed.WithDriverName(Name)
	}
	return nil
}
