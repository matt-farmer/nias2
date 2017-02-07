// distributor.go

// distributor creates parallel service access on demand
// creates a pool of listeners on a nats q to manage requests in parallel
// requests are routed sequentially through the service chain provided
// in NiasMessage.Route
package sms

import (
	//"github.com/nats-io/go-nats-streaming"
	"github.com/nsip/nias2/lib"
	//"log"
	"strings"
)

// creates a pool of message handlers which process the
// routing slip of each message thru the listed services
type Distributor struct{}

// uses single process for all streaming updates
// given the write-heavy nature of storage activities
// this is significantly faster than shared read/write
// services in parallel
func (vd *Distributor) Run(poolsize int, nats_cfg lib.NATSConfig) {

	config := lib.LoadDefaultConfig()
	ec := lib.CreateNATSConnection(nats_cfg)
	vs := NewStore()
	tt := lib.NewTransactionTracker(config.TxReportInterval, nats_cfg)

	for i := 0; i < poolsize; i++ {

		// create service handler
		go func(vs *Store, tt *lib.TransactionTracker) {
			sr := NewServiceRegister()
			ec.QueueSubscribe(lib.REQUEST_TOPIC, "distributor", func(m *lib.NiasMessage) {
				responses := sr.ProcessByRoute(m)
				for _, response := range responses {
					r := response
					ec.Publish(lib.STORE_TOPIC, r)
				}
				tt.IncrementTracker(m.TxID)
				// get status of transaction and add message to stream
				// if a notable status change has occurred
				sigChange, msg := tt.GetStatusReport(m.TxID)
				if sigChange {
					ec.Publish(lib.STORE_TOPIC, msg)
				}
			})

		}(vs, tt)

	}

	// create storage handler
	go func(vs *Store) {
		ec.Subscribe(lib.STORE_TOPIC, func(m *lib.NiasMessage) {
			if strings.HasPrefix(m.Target, SIF_MEMORY_STORE_PREFIX) {
				vs.StoreGraph(m)
			} else {
				vs.StoreMessage(m)
			}
		})

	}(vs)

}

// Following methods are all legacy
/*
// Use STAN as message bus
func (d *Distributor) RunSTANBus(poolsize int) {

	for i := 0; i < poolsize; i++ {

		sr := NewServiceRegister()
		pc := lib.NewSTANProcessChain()
		ms := NewMessageStore()

		// create storage handler
		go func(pc lib.STANProcessChain, ms *MessageStore, id int) {

			pc.store_in_conn.Subscribe(pc.store_in_subject, func(m *stan.Msg) {
				msg := DecodeNiasMessage(m.Data)
				if strings.HasPrefix(msg.Target, SIF_MEMORY_STORE_PREFIX) {
					ms.StoreGraph(msg)
				} else {
					ms.StoreMessage(msg)
				}
			})

		}(pc, ms, i)

		// create service handler
		go func(pc lib.STANProcessChain, sr *ServiceRegister, ms *MessageStore, id int) {

			pc.srvc_in_conn.Subscribe(pc.srvc_in_subject, func(m *stan.Msg) {
				msg := DecodeNiasMessage(m.Data)
				responses := sr.ProcessByRoute(msg)
				for _, response := range responses {
					r := response
					pc.srvc_out_conn.Publish(pc.store_in_subject, EncodeNiasMessage(&r))
				}
				ms.IncrementTracker(msg.TxID)
			})

		}(pc, sr, ms, i)

		// create an inbound handler to multiplex validation requests, create last or will drop messages
		go func(pc lib.STANProcessChain, id int) {

			pc.dist_in_conn.QueueSubscribe(pc.dist_in_subject, "distributor", func(m *stan.Msg) {
				pc.dist_out_conn.Publish(pc.dist_out_subject, m.Data)
			})

		}(pc, i)

	}
}

// alternate NATS Bus
// uses single process for all database updates
// given the write-heavy nature of storage activities
// this is significantly faster than shared read/write
// services in parallel
func (d *Distributor) RunNATSBus2(poolsize int) {

	ec := lib.CreateNATSConnection()
	vs := NewValidationStore()
	tt := NewTransactionTracker()

	for i := 0; i < poolsize; i++ {

		// create service handler
		go func(vs *ValidationStore, tt *TransactionTracker) {
			sr := NewServiceRegister()
			ec.QueueSubscribe(REQUEST_TOPIC, "distributor", func(m *NiasMessage) {
				responses := sr.ProcessByRoute(m)
				for _, response := range responses {
					r := response
					ec.Publish(STORE_TOPIC, r)
				}
				tt.IncrementTracker(m.TxID)
				// get status of transaction and add message to stream
				// if a notable status change has occurred
				sigChange, msg := tt.GetStatusReport(m.TxID)
				if sigChange {
					ec.Publish(STORE_TOPIC, msg)
				}
			})

		}(vs, tt)

	}

	// create storage handler
	go func(vs *ValidationStore) {

		ec.Subscribe(STORE_TOPIC, func(m *NiasMessage) {
			//log.Printf("%s %s", m.Target, "in")
			// if strings.HasPrefix(m.Target, SIF_MEMORY_STORE_PREFIX) {
			// ms.StoreGraph(m)
			// } else {
			vs.StoreMessage(m)
			// }
		})

	}(vs)

}

// Use regular NATS as message bus
func (d *Distributor) RunNATSBus(poolsize int) {

	for i := 0; i < poolsize; i++ {

		sr := NewServiceRegister()
		pc := NewNATSProcessChain()
		ms := NewMessageStore()

		// create storage handler
		go func(pc NATSProcessChain, ms *MessageStore) {

			pc.store_in_conn.Subscribe(pc.store_in_subject, func(m *NiasMessage) {
				if strings.HasPrefix(m.Target, SIF_MEMORY_STORE_PREFIX) {
					ms.StoreGraph(m)
				} else {
					ms.StoreMessage(m)
				}
			})

		}(pc, ms)

		// create service handler
		go func(pc NATSProcessChain, sr *ServiceRegister, ms *MessageStore) {

			pc.srvc_in_conn.Subscribe(pc.srvc_in_subject, func(m *NiasMessage) {
				responses := sr.ProcessByRoute(m)
				for _, response := range responses {
					r := response
					pc.srvc_out_conn.Publish(pc.store_in_subject, r)
				}
				ms.IncrementTracker(m.TxID)
			})

		}(pc, sr, ms)

		// create an inbound handler to multiplex validation requests, create last or will drop messages
		go func(pc NATSProcessChain) {

			pc.dist_in_conn.QueueSubscribe(pc.dist_in_subject, "distributor", func(m *NiasMessage) {
				pc.dist_out_conn.Publish(pc.dist_out_subject, m)
			})

		}(pc)

	}
}

// use internal channels as message bus
func (d *Distributor) RunMemBus(poolsize int) {

	for i := 0; i < poolsize; i++ {

		sr := NewServiceRegister()
		pc := NewMemProcessChain()
		ms := NewMessageStore()

		// create storage handler
		go func(pc MemProcessChain, ms *MessageStore) {

			var msg NiasMessage
			for {
				msg = <-pc.store_chan
				if msg.Target == SIF_MEMORY_STORE_PREFIX {
					ms.StoreGraph(&msg)
				} else {
					ms.StoreMessage(&msg)
				}
				//log.Printf("\t>%v %s %s\n", msg.Target, msg.MsgID, msg.SeqNo)
			}

		}(pc, ms)

		// create service handler
		go func(pc MemProcessChain, sr *ServiceRegister, ms *MessageStore) {

			for {
				msg := <-pc.req_chan
				// log.Printf("\t\tservice handler recieved msg: %+v", msg)
				responses := sr.ProcessByRoute(&msg)
				for _, response := range responses {
					r := response
					pc.store_chan <- r
				}
				ms.IncrementTracker(msg.TxID)
			}

		}(pc, sr, ms)

	}

}
*/

//
//
//
//
//
//
//
//
//
//
//
