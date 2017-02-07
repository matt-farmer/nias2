// serviceregister.go
package sms

// simple hashtable of service handlers stored by name
// will be matched against required tasks in NasMessage.Route meta-data

import (
	"github.com/nsip/nias2/lib"
	"log"
	"sync"
)

// simple thread-safe container for group of services that will be available
// to process messages passed from a distributor node
type ServiceRegister struct {
	sync.RWMutex
	registry map[string]lib.NiasService
}

// creates a ServiceRegister with properly initilaised internal map
// processing services are stored with a name and the referenced NiasService
func NewServiceRegister() *ServiceRegister {
	return createDefaultServiceRegister()
}

// add a service to the registry with a name
func (sr *ServiceRegister) AddService(servicename string, service lib.NiasService) {
	sr.Lock()
	sr.registry[servicename] = service
	sr.Unlock()
}

// remove a service from the registry by name
func (sr *ServiceRegister) RemoveService(servicename string) {
	sr.Lock()
	delete(sr.registry, servicename)
	sr.Unlock()
}

// return a service by providing the name
func (sr *ServiceRegister) FindService(servicename string) lib.NiasService {
	sr.RLock()
	defer sr.RUnlock()
	return sr.registry[servicename]
}

// build register with default set of services
func createDefaultServiceRegister() *ServiceRegister {

	log.Println("Creating services & register")
	sr := ServiceRegister{}
	sr.registry = make(map[string]lib.NiasService)

	priv1, err := NewPrivacyService()
	if err != nil {
		log.Fatal("Unable to create privacy service ", err)
	}

	s2g1, err := NewSif2GraphService()
	if err != nil {
		log.Fatal("Unable to create sif2graph service ", err)
	}

	/*
		sif, err := NewSifValidationService()
		if err != nil {
			log.Fatal("Unable to create numeric validation service ", err)
		}
	*/

	sr.AddService("privacy", priv1)
	sr.AddService("sif2graph", s2g1)
	//sr.AddService("sifvalidation", sif)

	log.Println("services created & installed in register")

	return &sr

}

func (sr *ServiceRegister) ProcessByRoute(m *lib.NiasMessage) []lib.NiasMessage {

	response_msgs := make([]lib.NiasMessage, 0)

	route := m.Route

	// log.Printf("\t\tservice register recieved msg: %+v", m)

	for _, sname := range route {

		// retrieve service from registry & execute
		srvc := sr.FindService(sname)
		responses, err := srvc.HandleMessage(m)
		if err != nil {
			log.Println("\t *** got an error on service handler " + sname + " ***")
			log.Println("\t", err)
		} else {
			// pass the responses to the message store
			// log.Printf("\t\tservice %s returned %d responses: %+v", sname, len(responses), responses)
			for _, r := range responses {
				response := r
				response.Source = sname
				response_msgs = append(response_msgs, response)
			}
		}
	}

	// log.Printf("\t\tresponse messages: %+v", response_msgs)
	return response_msgs

}
