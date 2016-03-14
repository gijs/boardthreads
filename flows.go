package main

import (
	"bt/db"
	"bt/mailgun"

	log "github.com/Sirupsen/logrus"
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
