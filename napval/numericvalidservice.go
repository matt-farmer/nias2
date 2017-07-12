// numericvalidservice.go
package napval

import (
	"fmt"
	"github.com/nsip/nias2/lib"
	"github.com/nsip/nias2/xml"
	"log"
	"strconv"
)

// numeric value validator service

// implementation of the id service
type NumericValidService struct {
}

// create a new id service instance
func NewNumericValidService() (*NumericValidService, error) {
	num := NumericValidService{}

	return &num, nil
}

// implement the nias Service interface
func (num *NumericValidService) HandleMessage(req *lib.NiasMessage) ([]lib.NiasMessage, error) {

	responses := make([]lib.NiasMessage, 0)

	rr, ok := req.Body.(xml.RegistrationRecord)
	if !ok {
		log.Println("NumericValidService received a message that is not a RegistrationRecord, ignoring")
		return responses, nil
	}

	fte := rr.FTE
	desc := ""
	field := "FTE"
	ok = true

	if len(fte) > 0 {
		fte_f, err := strconv.ParseFloat(fte, 64)
		if err != nil {
			ok = false
			desc = fmt.Sprintf("FTE %s is not a well formed float", fte)
		} else {
			if fte_f < 0 {
				ok = false
				desc = "FTE is less than 0"
			}
			if fte_f > 1 {
				ok = false
				desc = "FTE is greater than 1"
			}
		}
	}
	if !ok {
		responses = add_error(responses, desc, field, req)
	}

	return responses, nil

}

func add_error(responses []lib.NiasMessage, desc string, field string, req *lib.NiasMessage) []lib.NiasMessage {
	ve := ValidationError{
		Description:  desc,
		Field:        field,
		OriginalLine: req.SeqNo,
		Vtype:        "numeric",
		Severity:     "error",
	}
	r := lib.NiasMessage{}
	r.TxID = req.TxID
	r.SeqNo = req.SeqNo
	// r.Target = VALIDATION_PREFIX
	r.Body = ve
	responses = append(responses, r)
	return responses
}
