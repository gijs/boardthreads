package main

import (
	"bt/db"
	"bt/mailgun"
	"bt/paypal"
	"bt/trello"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/segmentio/analytics-go"
)

func MaybeDeleteDomainAndRouteFlow(oldAddress *db.Address, newOutboundAddr string) {
	if oldAddress == nil || oldAddress.DomainName == "" {
		return
	}

	addresses, err := db.ListAddressesOnDomain(oldAddress.DomainName)
	if err != nil {
		log.WithFields(log.Fields{
			"err":    err.Error(),
			"domain": oldAddress.DomainName,
		}).Warn("couldn't get addresses on domain")
		return
	}

	// delete domain from mailgun if it is not being used by any other address
	if len(addresses) == 0 || (len(addresses) == 1 && oldAddress.OutboundAddr == addresses[0]) {
		log.WithFields(log.Fields{
			"domain":            oldAddress.DomainName,
			"addressesondomain": addresses,
			"oldaddress":        oldAddress.OutboundAddr,
		}).Info("will delete domain")
		err = mailgun.Client.DeleteDomain(oldAddress.DomainName)
		if err != nil {
			log.WithFields(log.Fields{
				"err":    err.Error(),
				"domain": oldAddress.DomainName,
			}).Warn("failed to delete oldAddress domain")
		}
	}

	// delete routeId from mailgun and DB if the outboundaddr here has changed
	if oldAddress.OutboundAddr != newOutboundAddr && oldAddress.RouteId != "" {
		log.WithFields(log.Fields{
			"routeId": oldAddress.RouteId,
			"address": oldAddress.OutboundAddr,
		}).Info("will delete route")
		err = mailgun.Client.DeleteRoute(oldAddress.RouteId)
		if err != nil {
			log.WithFields(log.Fields{
				"err":     err.Error(),
				"routeId": oldAddress.RouteId,
			}).Warn("failed to delete route")
		}
	}
	return
}

func MaybeFillDomainInformation(address *db.Address) {
	if address.DomainName != "" {
		domain, err := mailgun.GetDomain(address.DomainName)
		if err != nil {
			log.WithFields(log.Fields{
				"name": address.DomainName,
				"err":  err.Error(),
			}).Warn("failed to fetch domain from mailgun")
			return
		}
		address.DomainStatus = domain
	}
}

func MaybeDowngradeAddress(address *db.Address) error {
	if address.PaypalProfileId == "" {
		return nil
	}

	err := paypal.DeleteSubscription(address.PaypalProfileId)
	if err != nil {
		log.WithFields(log.Fields{
			"err":             err.Error(),
			"address":         address.InboundAddr,
			"paypalProfileId": address.PaypalProfileId,
		}).Warn("couldn't cancel paypal subscription")
		return nil
	}

	err = db.RemovePaypalProfileId(address.InboundAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"err":             err.Error(),
			"address":         address.InboundAddr,
			"paypalProfileId": address.PaypalProfileId,
		}).Error("failed to remove paypalProfileId from address")
		return err
	}

	// tracking
	segment.Track(&analytics.Track{
		Event:  "Cancelled subscription",
		UserId: address.UserId,
		Properties: map[string]interface{}{
			"address":  address.InboundAddr,
			"provider": "Paypal",
		},
	})

	return nil
}

func CommentWithNewSendingParams(cardId string) {
	logger := log.WithField("card", cardId)

	// new params, just for commenting with the updated data
	params, err := db.GetEmailParamsForCard(cardId)
	if err != nil {
		logger.WithField("err", err).Warn("couldn't get sending params for the card")
	}

	// fetch this card from trello
	card, err := trello.Client.Card(cardId)
	if err != nil {
		logger.WithField("err", err).Warn("couldn't find the card on trello, will not do anything")
		return
	}

	// then post a comment
	_, err = card.AddComment(
		fmt.Sprintf(
			`All comments from now on will be sent to **%s** with the subject _%s_.`,
			params.Recipients,
			params.LastMailSubject,
		),
	)
	if err != nil {
		logger.WithField("err", err).Warn("couldn't post comment with new card params.")
	}
}
