package naprr

import (
	"log"
	"sync"
)

// var sr = NewStreamReader()
// var rg = NewReportGenerator()

type ReportBuilder struct {
	sr *StreamReader
	rg *ReportGenerator
}

func NewReportBuilder() *ReportBuilder {
	return &ReportBuilder{sr: NewStreamReader(), rg: NewReportGenerator()}
}

func (rb *ReportBuilder) Run() {

	var wg sync.WaitGroup

	schools := rb.sr.GetSchoolDetails()
	nd := rb.sr.GetNAPLANData(META_STREAM)

	for _, subslice := range schools {
		for _, school := range subslice {
			wg.Add(1)
			go rb.createSchoolReports(nd, school.ACARAId, &wg)
		}
	}

	wg.Add(1)
	go rb.createTestReports(nd, &wg)

	// block until all reports generated
	wg.Wait()
	log.Println("All reports generated")

}

// Year 3 Writing
func (rb *ReportBuilder) RunYr3W(schools bool, student_ids map[string]string, NaprrConfig naprr_config) {

	var wg sync.WaitGroup

	log.Printf("%v\n", student_ids)
	log.Println("Getting student data")
	sr := rb.sr.GetStudentAndResultsData(student_ids, NaprrConfig)
	log.Println("Gotten student data")
	nd := rb.sr.GetNAPLANData(META_YR3W_STREAM)

	if schools {
		schoolslist := rb.sr.GetSchoolDetails()
		for _, subslice := range schoolslist {
			for _, school := range subslice {
				wg.Add(1)
				go rb.createSchoolReports(nd, school.ACARAId, &wg)
			}
		}
	}

	wg.Add(1)
	go rb.createYr3WReports(nd, sr, &wg)

	// block until all reports generated
	wg.Wait()
	log.Println("All Year 3 Writing report data generated")

}

// generate school-level data reports
func (rb *ReportBuilder) createSchoolReports(nd *NAPLANData, acaraid string, wg *sync.WaitGroup) {
	sd := rb.sr.GetSchoolData(acaraid)
	rb.rg.GenerateParticipationData(nd, sd)
	log.Println("Participation data created for: ", acaraid)
	rb.rg.GenerateSchoolScoreSummaryData(nd, sd)
	log.Println("Score summary data created for: ", acaraid)
	rb.rg.GenerateDomainScoreData(nd, sd)
	log.Println("Domain scores data created for: ", acaraid)

	wg.Done()
}

// generate test-level reports
func (rb *ReportBuilder) createTestReports(nd *NAPLANData, wg *sync.WaitGroup) {
	rb.rg.GenerateCodeFrameData(nd)
	log.Println("Codeframe data created.")
	wg.Done()
}

// generate test-level reports
func (rb *ReportBuilder) createYr3WReports(nd *NAPLANData, sr *StudentAndResultsData, wg *sync.WaitGroup) {
	rb.rg.GenerateYr3WData(nd, sr)
	log.Println("Year 3 Writing XML data created.")
	wg.Done()
}
