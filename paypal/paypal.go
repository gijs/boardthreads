package paypal

import (
	"encoding/json"
	"os/exec"
)

func GetAuthURL(userId, address string) (string, error) {
	command := exec.Command("./authenticate")
	stdin, err := command.StdinPipe()
	if err != nil {
		return "", err
	}

	arguments, err := json.Marshal(struct {
		SuccessCallback       string `json:"RETURNURL"`
		ErrorCallback         string `json:"CANCELURL"`
		Amount                int    `json:"PAYMENTREQUEST_0_AMT"`
		BrandName             string `json:"BRANDNAME"`
		Description           string `json:"L_BILLINGAGREEMENTDESCRIPTION0"`
		BuyerEmailOptinEnable int    `json:"BUYEREMAILOPTINENABLE"`
	}{"/billing/paypal/success", "/billing/paypal/failure", 10, "Boardthreads Helpdesk", "boardthreads.com subscription for user" + userId, 0})
	if err != nil {
		return "", err
	}

	stdin.Write(arguments)
	urlBytes, err := command.Output()
	return string(urlBytes), err
}

func CreateSubscription(userId string) error {
	command := exec.Command("./create-subscription")
	stdin, err := command.StdinPipe()
	if err != nil {
		return err
	}

	arguments, err := json.Marshal(struct {
		Amount              int    `json:"AMT"`
		Description         string `json:"DESC"`
		BillingPeriod       string `json:"BILLINGPERIOD"`
		BillingFrequency    int    `json:"BILLINGFREQUENCY"`
		MaxFailed           int    `json:"MAXFAILEDPAYMENTS"`
		AutoBillOutstanding string `json:"AUTOBILLOUTAMT"`
	}{10, "boardthreads.com subscription for user" + userId, "Month", 1, 3, "AddToNextBilling"})
	if err != nil {
		return err
	}

	stdin.Write(arguments)
	_, err = command.Output()
	return err
}
