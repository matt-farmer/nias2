// report-resolvers.go

//
// resolver logic for composite reporting objects
//
package naprrql

import (
	"github.com/nsip/nias2/naprr"
	"github.com/nsip/nias2/xml"
	"github.com/playlyfe/go-graphql"
)

// helper type for summary
type ParticipationSummary struct {
	Domain            string
	ParticipationCode string
}

// reporting object for student participation
type ParticipationDataSet struct {
	Student    xml.RegistrationRecord
	School     xml.SchoolInfo
	EventInfos []naprr.EventInfo
	Summary    []ParticipationSummary
}

func buildReportResolvers() map[string]interface{} {

	resolvers := map[string]interface{}{}

	//
	// shorthand lookup objects for basic school info
	//
	resolvers["NaplanData/school_details"] = func(params *graphql.ResolveParams) (interface{}, error) {
		return getObjects(getIdentifiers("SchoolDetails"))
	}

	//
	// resolver for score summary report object
	//
	resolvers["NaplanData/score_summary_report_by_school"] = func(params *graphql.ResolveParams) (interface{}, error) {

		// get the acara ids from the request params
		acaraids := make([]string, 0)
		for _, a_id := range params.Args["acaraIDs"].([]interface{}) {
			acaraid, _ := a_id.(string)
			acaraids = append(acaraids, acaraid)
		}

		// get the sif refid for each of the acarids supplied
		refids := make([]string, 0)
		for _, acaraid := range acaraids {
			refid := getIdentifiers(acaraid)[0]
			refids = append(refids, refid)
		}

		// now construct the composite keys
		school_summary_keys := make([]string, 0)
		for _, refid := range refids {
			school_summary_keys = append(school_summary_keys, refid+":NAPTestScoreSummary")
		}

		summ_refids := make([]string, 0)
		for _, summary_key := range school_summary_keys {
			ids := getIdentifiers(summary_key)
			for _, id := range ids {
				summ_refids = append(summ_refids, id)
			}
		}

		summaries, err := getObjects(summ_refids)
		summary_datasets := make([]naprr.ScoreSummaryDataSet, 0)
		for _, summary := range summaries {
			summ, _ := summary.(xml.NAPTestScoreSummary)
			testid := []string{summ.NAPTestRefId}
			obj, _ := getObjects(testid)
			test, _ := obj[0].(xml.NAPTest)
			sds := naprr.ScoreSummaryDataSet{Summ: summ, Test: test}
			summary_datasets = append(summary_datasets, sds)
		}

		return summary_datasets, err

	}

	resolvers["NaplanData/school_infos_by_acaraid"] = func(params *graphql.ResolveParams) (interface{}, error) {

		// get the acara ids from the request params
		acaraids := make([]string, 0)
		for _, a_id := range params.Args["acaraIDs"].([]interface{}) {
			acaraid, _ := a_id.(string)
			acaraids = append(acaraids, acaraid)
		}

		// get the sif refid for each of the acarids supplied
		refids := make([]string, 0)
		for _, acaraid := range acaraids {
			refid := getIdentifiers(acaraid)[0]
			refids = append(refids, refid)
		}

		// get the school infos from the datastore
		siObjects, err := getObjects(refids)
		schoolInfos := make([]xml.SchoolInfo, 0)
		for _, sio := range siObjects {
			si, _ := sio.(xml.SchoolInfo)
			schoolInfos = append(schoolInfos, si)
		}

		return schoolInfos, err

	}

	resolvers["NaplanData/students_by_school"] = func(params *graphql.ResolveParams) (interface{}, error) {

		// get the acara ids from the request params
		acaraids := make([]string, 0)
		for _, a_id := range params.Args["acaraIDs"].([]interface{}) {
			acaraid, _ := a_id.(string)
			acaraids = append(acaraids, acaraid)
		}

		// get students for the schools
		studentids := make([]string, 0)
		for _, acaraid := range acaraids {
			key := "student_by_acaraid:" + acaraid
			studentRefIds := getIdentifiers(key)
			studentids = append(studentids, studentRefIds...)
		}

		return getObjects(studentids)

	}

	resolvers["NaplanData/domain_scores_report_by_school"] = func(params *graphql.ResolveParams) (interface{}, error) {

		// get the acara ids from the request params
		acaraids := make([]string, 0)
		for _, a_id := range params.Args["acaraIDs"].([]interface{}) {
			acaraid, _ := a_id.(string)
			acaraids = append(acaraids, acaraid)
		}

		// get students for the schools
		studentids := make([]string, 0)
		for _, acaraid := range acaraids {
			key := "student_by_acaraid:" + acaraid
			studentRefIds := getIdentifiers(key)
			studentids = append(studentids, studentRefIds...)
		}

		// get responses for student
		responseids := make([]string, 0)
		for _, studentid := range studentids {
			key := "responseset_by_student:" + studentid
			responseRefId := getIdentifiers(key)
			responseids = append(responseids, responseRefId...)
		}

		// get responses
		responses, err := getObjects(responseids)
		if err != nil {
			return []interface{}{}, err
		}

		// construct RDS by including referenced test
		results := make([]naprr.ResponseDataSet, 0)
		for _, response := range responses {
			resp, _ := response.(xml.NAPResponseSet)
			// domain score entries will be null if response not completed e.g. abandoned
			if resp.DomainScore.RawScore == "" {
				continue
			}
			tests, err := getObjects([]string{resp.TestID})
			test, ok := tests[0].(xml.NAPTest)
			if err != nil || !ok {
				return []interface{}{}, err
			}
			rds := naprr.ResponseDataSet{Test: test, Response: resp}
			results = append(results, rds)
		}

		return results, nil

	}

	resolvers["NaplanData/participation_report_by_school"] = func(params *graphql.ResolveParams) (interface{}, error) {

		// get the acara ids from the request params
		acaraids := make([]string, 0)
		for _, a_id := range params.Args["acaraIDs"].([]interface{}) {
			acaraid, _ := a_id.(string)
			acaraids = append(acaraids, acaraid)
		}
		// log.Printf("acara-ids: \n\n %#v\n\n", acaraids)

		// get students for the schools
		studentids := make([]string, 0)
		for _, acaraid := range acaraids {
			key := "student_by_acaraid:" + acaraid
			studentRefIds := getIdentifiers(key)
			studentids = append(studentids, studentRefIds...)
		}
		// log.Printf("studentids: \n\n %#v\n\n", studentids)
		studentObjs, err := getObjects(studentids)
		if err != nil {
			return []interface{}{}, err
		}
		// log.Printf("\n\n no students objects: %d", len(studentObjs))

		// iterate students and assemble ParticipationDataSets
		results := make([]ParticipationDataSet, 0)
		for _, studentObj := range studentObjs {
			student, _ := studentObj.(xml.RegistrationRecord)
			studentEventIds := getIdentifiers(student.RefId + ":NAPEventStudentLink")
			// log.Printf("\n\n student event ids: \n\n%#v\n\n", studentEventIds)
			eventObjs, err := getObjects(studentEventIds)
			if err != nil {
				return []interface{}{}, err
			}
			eventInfos := make([]naprr.EventInfo, 0)
			for _, eventObj := range eventObjs {
				event := eventObj.(xml.NAPEvent)
				// log.Printf("\n\n   event: \n\n%#v\n\n", event)
				testObj, err := getObjects([]string{event.TestID})
				if err != nil {
					return []interface{}{}, err
				}
				test := testObj[0].(xml.NAPTest)
				// log.Printf("\n\n   test: \n\n%#v\n\n", test)
				eventInfo := naprr.EventInfo{Test: test, Event: event}
				eventInfos = append(eventInfos, eventInfo)
			}
			// log.Printf("\n\n   eventinfos: \n\n%#v\n\n", eventInfos)
			schoolKey := eventInfos[0].Event.SchoolRefID
			schoolObj, err := getObjects([]string{schoolKey})
			if err != nil {
				return []interface{}{}, err
			}
			school, _ := schoolObj[0].(xml.SchoolInfo)
			// log.Printf("\n\n   school: \n\n%#v\n\n", school)
			// construct summary
			summaries := make([]ParticipationSummary, 0)
			for _, event := range eventInfos {
				summary := ParticipationSummary{
					Domain:            event.Test.TestContent.TestDomain,
					ParticipationCode: event.Event.ParticipationCode}
				summaries = append(summaries, summary)
			}
			pds := ParticipationDataSet{Student: student,
				School:     school,
				EventInfos: eventInfos,
				Summary:    summaries,
			}
			// log.Printf("\n\n   pds: \n\n%#v\n\n", pds)
			results = append(results, pds)
		}

		return results, nil

	}

	resolvers["NaplanData/codeframe_report"] = func(params *graphql.ResolveParams) (interface{}, error) {
		// get the codeframe objects
		codeframes := make([]xml.NAPCodeFrame, 0)
		codeframeIds := getIdentifiers("NAPCodeFrame")
		codeFrameObjs, err := getObjects(codeframeIds)
		if err != nil {
			return []interface{}{}, err
		}
		for _, codeframeObj := range codeFrameObjs {
			codeFrame, _ := codeframeObj.(xml.NAPCodeFrame)
			codeframes = append(codeframes, codeFrame)
		}

		cfds := make([]naprr.CodeFrameDataSet, 0)
		for _, codeframe := range codeframes {
			testObj, err := getObjects([]string{codeframe.NAPTestRefId})
			if err != nil {
				return []interface{}{}, err
			}
			test, _ := testObj[0].(xml.NAPTest)
			for _, cf_testlet := range codeframe.TestletList.Testlet {
				tlObj, _ := getObjects([]string{cf_testlet.NAPTestletRefId})
				if err != nil {
					return []interface{}{}, err
				}
				tl, _ := tlObj[0].(xml.NAPTestlet)
				for _, cf_item := range cf_testlet.TestItemList.TestItem {
					tiObj, err := getObjects([]string{cf_item.TestItemRefId})
					if err != nil {
						return []interface{}{}, err
					}
					ti, _ := tiObj[0].(xml.NAPTestItem)
					// log.Printf("\t\t%s", ti.TestItemContent.ItemName)
					cfd := naprr.CodeFrameDataSet{
						Test:    test,
						Testlet: tl,
						Item:    ti,
					}
					cfds = append(cfds, cfd)
				}
			}
		}

		return cfds, nil

	}

	return resolvers
}
