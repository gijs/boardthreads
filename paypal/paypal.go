package paypal

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
)

var here string

func init() {
	if os.Getenv("GOPATH") != "" {
		here = filepath.Join(os.Getenv("GOPATH"), "src/bt/paypal")
	} else {
		log.Info("no GOPATH found.")
		var err error
		here, err = filepath.Abs("./paypal")
		if err != nil {
			log.Fatal(err)
		}
	}
}

func GetAuthURL(userId, address, successURL, failureURL string) (string, error) {
	arguments, err := json.Marshal(struct {
		SuccessCallback       string `json:"RETURNURL"`
		ErrorCallback         string `json:"CANCELURL"`
		Amount                int    `json:"PAYMENTREQUEST_0_AMT"`
		BrandName             string `json:"BRANDNAME"`
		Description           string `json:"L_BILLINGAGREEMENTDESCRIPTION0"`
		BuyerEmailOptinEnable int    `json:"BUYEREMAILOPTINENABLE"`
	}{successURL, failureURL, 18, "Boardthreads Helpdesk", descPrefix + address, 0})
	if err != nil {
		return "", err
	}

	command := exec.Command(filepath.Join(here, "authenticate"), string(arguments))

	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

func CreateSubscription(userId, address, token, payerId string) (profileId string, err error) {
	arguments, err := json.Marshal(struct {
		Amount              int    `json:"AMT"`
		Description         string `json:"DESC"`
		BillingPeriod       string `json:"BILLINGPERIOD"`
		BillingFrequency    int    `json:"BILLINGFREQUENCY"`
		MaxFailed           int    `json:"MAXFAILEDPAYMENTS"`
		AutoBillOutstanding string `json:"AUTOBILLOUTAMT"`
	}{18, descPrefix + address, "Month", 1, 3, "AddToNextBilling"})
	if err != nil {
		return "", err
	}

	command := exec.Command(filepath.Join(here, "create-subscription"), token, payerId, string(arguments))

	output, err := command.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

const descPrefix string = "boardthreads.com subscription for address "

func DeleteSubscription(profileId string) error {
	command := exec.Command(filepath.Join(here, "delete-subscription"), profileId)

	output, err := command.CombinedOutput()
	if err != nil {
		log.Debug(string(output))
		return err
	}

	return nil
}
