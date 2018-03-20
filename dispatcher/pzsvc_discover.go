package main

import (
	"errors"
	"time"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

type pzSvcDiscoverer struct {
}

func newPzSvcDiscoverer() *pzSvcDiscoverer {
	return &pzSvcDiscoverer{}
}

func (d pzSvcDiscoverer) discoverSvcID(s *pzsvc.Session, config *pzsvc.Config) (svcID string, err error) {
	// Check for the Service ID. If it exists, then grab the ID. If it doesn't exist, then Register it.
	svcID, err = pzsvc.FindMySvc(*s, config.SvcName)
	if err != nil {
		pzsvc.LogSimpleErr(*s, "Dispatcher could not find Piazza Service ID.  Initial Error: ", err)
		return
	}
	if svcID != "" {
		return
	}

	// If no Service ID is found, attempt to register it.
	pzsvc.LogInfo(*s, "Could not find service.  Will attempt to register it.")
	_, newSession := pzsvc.ParseConfigAndRegister(*s, config)

	// With registration completed, Check back for Service ID
	time.Sleep(time.Duration(1) * time.Second)
	svcID, err = pzsvc.FindMySvc(newSession, config.SvcName)
	if err != nil {
		pzsvc.LogSimpleErr(*s, "Dispatcher could not find new Service ID post registration.  Initial Error: ", err)
		return
	}
	if svcID == "" {
		pzsvc.LogInfo(*s, "Could not find service ID post registration. The application cannot start. Please verify Service Registration and restart the application.")
		err = errors.New("could not find service ID post registration")
	}
	return
}
