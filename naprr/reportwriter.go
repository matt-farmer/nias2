package naprr

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"text/template"
)

// var rw_sr = NewStreamReader()
// var t *template.Template

type ReportWriter struct {
	sr *StreamReader
	t  *template.Template
}

func NewReportWriter() *ReportWriter {
	return &ReportWriter{sr: NewStreamReader(), t: loadTemplates()}
}

func (rw *ReportWriter) Run() {

	// loadTemplates()

	schools := rw.sr.GetSchoolDetails()

	rw.writeSchoolLevelReports(schools)
	rw.writeAggregateSchoolReports(schools)
	rw.writeTestLevelReports()

	log.Println("All reports written\n")
}

// create data reports from the test strucutre
func (rw *ReportWriter) WriteYr3WReports() {

	log.Println("Creating Year 3 Writing XML...")

	var wg sync.WaitGroup

	nd := rw.sr.GetNAPLANData(REPORTS_YR3W)
	rbs := rw.sr.GetResultsByStudent()

	wg.Add(1)

	go rw.writeYr3WritingReport(nd, rbs, &wg)

	wg.Wait()

	log.Println("Year 3 Writing XML created.")
}

// create data reports from the test strucutre
func (rw *ReportWriter) writeTestLevelReports() {

	log.Println("Creating test-level reports...")

	var wg sync.WaitGroup

	cfds := rw.sr.GetCodeFrameData(REPORTS_CODEFRAME)
	nd := rw.sr.GetNAPLANData(META_STREAM)

	wg.Add(3)

	go rw.writeCodeFrameReportXML(nd, &wg)
	go rw.writeCodeFrameReport(cfds, &wg)
	go rw.writeCodeFrameWritingReport(cfds, &wg)

	wg.Wait()

	log.Println("Test-level reports created.")
}

// create data reports for each school
func (rw *ReportWriter) writeSchoolLevelReports(schools [][]SchoolDetails) {

	var wg sync.WaitGroup

	log.Println("Creating school-level reports...")

	for _, subslice := range schools {
		for _, school := range subslice {
			wg.Add(1)
			go rw.writeSchoolReports(school.ACARAId, &wg)
		}
	}

	wg.Wait()

	log.Println("School-level reports created.")
}

// create aggregate reports from school-level data
// uses file-concat for speed and to manage no. open connections & filehandles
// esp. on eg win32 environment
func (rw *ReportWriter) writeAggregateSchoolReports(schools [][]SchoolDetails) {

	log.Println("Creating aggregate reports...")

	outputPath := "out/"

	//report types we want to aggregate
	reportTypes := []string{"participation", "score_summary", "domain_scores"}

	for _, reportType := range reportTypes {
		// create empty aggregate report file with header
		outputFile := rw.createSummaryFileWithHeader(reportType)
		for _, subslice := range schools {
			filePaths := make([]string, 0)
			for _, schoolDetails := range subslice {
				filePath := outputPath + schoolDetails.ACARAId + "/" + reportType + ".dat"
				// check whether the file exists, ignore if doesn't
				_, err := os.Stat(filePath)
				if err != nil {
					continue
				}
				filePaths = append(filePaths, filePath)
			}
			if len(filePaths) > 0 {
				concatenateFiles(filePaths, outputFile)
				// remove temp data files
				for _, file := range filePaths {
					err := os.Remove(file)
					if err != nil {
						fmt.Println("Unable to remove temp data file: ", file, err)
					}
				}
			}
		}
	}

	log.Println("Aggregate reports created.")

}

// load all output templates once at start-up
func loadTemplates() *template.Template {

	t := template.Must(template.ParseGlob("templates/*"))
	// log.Println(t.DefinedTemplates())
	return t
}

func (rw *ReportWriter) createSummaryFileWithHeader(reportType string) (filePath string) {

	fname := "out/" + reportType + ".csv"

	var tmpl *template.Template
	switch reportType {
	case "participation":
		tmpl = rw.t.Lookup("participation_hdr.tmpl")
	case "score_summary":
		tmpl = rw.t.Lookup("score_summary_hdr.tmpl")
	case "domain_scores":
		tmpl = rw.t.Lookup("domainscore_hdr.tmpl")
	}

	// remove any previous versions
	err := os.RemoveAll(fname)
	if err != nil {
		fmt.Println("Cannot delete previous aggregate file: ", fname)
	}

	aggregateFile, err := os.Create(fname)
	defer aggregateFile.Close()
	if err != nil {
		fmt.Println("Cannot open aggregate file: ", fname, err)
	}

	// write the header
	// doesn't actually need any data - all text fields so pass nil struct as data
	if err := tmpl.Execute(aggregateFile, nil); err != nil {
		fmt.Println("Cannot execute template header: ", reportType, err)
	}

	aggregateFile.Close()

	return fname

}

func (rw *ReportWriter) writeSchoolReports(acaraid string, wg *sync.WaitGroup) {

	rw.writeParticipationReport(acaraid)
	rw.writeScoreSummaryReport(acaraid)
	rw.writeDomainScoreReport(acaraid)
	rw.splitNAPXMLperSchool(acaraid)

	wg.Done()
}

// report of test structure for writing items only
// with extended item information
func (rw *ReportWriter) writeCodeFrameWritingReport(cfds []CodeFrameDataSet, wg *sync.WaitGroup) {

	thdr := rw.t.Lookup("codeframe_writing_hdr.tmpl")
	trow := rw.t.Lookup("codeframe_writing_row.tmpl")

	// create directory for the school
	fpath := "out/"
	err := os.MkdirAll(fpath, os.ModePerm)
	check(err)

	// create the report data file in the output directory
	// delete any ecisting files and create empty new one
	fname := fpath + "codeframe_writing.dat"
	err = os.RemoveAll(fname)
	f, err := os.Create(fname)
	check(err)
	defer f.Close()

	// write the data - writing items only
	for _, cfd := range cfds {
		if cfd.Test.TestContent.TestDomain == "Writing" {
			if err := trow.Execute(f, cfd); err != nil {
				check(err)
			}
		}
	}

	// write the empty header file
	fname2 := fpath + "codeframe_writing.csv"
	f2, err := os.Create(fname2)
	check(err)
	defer f2.Close()

	// doesn't actually need any data - all text fields so pass nil struct as data
	if err := thdr.Execute(f2, nil); err != nil {
		check(err)
	}

	inputFile := []string{fname}
	outputFile := fname2

	concatenateFiles(inputFile, outputFile)

	// remove the temp data files
	err = os.RemoveAll(fname)
	check(err)

	log.Printf("Codeframe writing report created for: %d elements", len(cfds))

	wg.Done()

}

// report of test structure for writing items only
// with extended item information
func (rw *ReportWriter) writeYr3WritingReport(nd *NAPLANData, rbs []ResultsByStudent, wg *sync.WaitGroup) {

	// create directory for the school
	fpath := "yr3w/"
	err := os.MkdirAll(fpath, os.ModePerm)
	check(err)

	// create the report data file in the output directory
	// delete any ecisting files and create empty new one
	fname := fpath + "codeframe_writing.xml"
	err = os.RemoveAll(fname)
	f, err := os.Create(fname)
	check(err)
	defer f.Close()

	e := xml.NewEncoder(f)
	e.Indent("", "  ")
	f.WriteString("<NAPResultsReporting>\n")
	cfcount := 0
	for _, val := range nd.Tests {
		e.Encode(val)
		cfcount++
	}
	for _, val := range nd.Testlets {
		e.Encode(val)
		cfcount++
	}
	for _, val := range nd.Items {
		e.Encode(val)
		cfcount++
	}
	for _, r := range rbs {
		e.Encode(r.Student)
		e.Encode(r.Event)
		e.Encode(r.ResponseSet)
	}

	e.Flush()
	f.WriteString("</NAPResultsReporting>\n")

	log.Printf("Codeframe writing report created for: %d codeframe elements and %d results elements", cfcount, len(rbs))

	wg.Done()

}

// report of test structure for writing items only
// with extended item information
func (rw *ReportWriter) writeCodeFrameReportXML(nd *NAPLANData, wg *sync.WaitGroup) {

	fpath := "out/"
	err := os.MkdirAll(fpath, os.ModePerm)
	check(err)

	// create the report data file in the output directory
	// delete any ecisting files and create empty new one
	fname := fpath + "codeframe.xml"
	err = os.RemoveAll(fname)
	f, err := os.Create(fname)
	check(err)
	defer f.Close()

	e := xml.NewEncoder(f)
	e.Indent("", "  ")
	f.WriteString("<NAPResultsReporting>\n")
	seen := make(map[string]bool)
	cfcount := 0
	for _, codeframe := range nd.Codeframes {
		if len(codeframe.NAPTestRefId) == 0 || seen[codeframe.NAPTestRefId] {
			continue
		}
		if _, ok := nd.Tests[codeframe.NAPTestRefId]; ok {
			e.Encode(nd.Tests[codeframe.NAPTestRefId])
			seen[codeframe.NAPTestRefId] = true
			cfcount++
		}
		for _, cf_testlet := range codeframe.TestletList.Testlet {
			if len(cf_testlet.NAPTestletRefId) == 0 /*|| seen[cf_testlet.NAPTestletRefId] */ {
				continue
			}
			if _, ok := nd.Testlets[cf_testlet.NAPTestletRefId]; ok {
				e.Encode(nd.Testlets[cf_testlet.NAPTestletRefId])
				seen[cf_testlet.NAPTestletRefId] = true
				cfcount++
			}
			for _, cf_item := range cf_testlet.TestItemList.TestItem {
				if len(cf_item.TestItemRefId) == 0 /* || seen[cf_item.TestItemRefId] */ {
					continue
				}
				if _, ok := nd.Items[cf_item.TestItemRefId]; ok {
					seen[cf_item.TestItemRefId] = true
					e.Encode(nd.Items[cf_item.TestItemRefId])
					cfcount++
				}
			}
		}
	}

	e.Flush()
	f.WriteString("</NAPResultsReporting>\n")

	log.Printf("Codeframe writing report in XML created for: %d codeframe elements", cfcount)

	wg.Done()

}

// report of test structure, is written only once
// as an aggrregate report, not at school level
func (rw *ReportWriter) writeCodeFrameReport(cfds []CodeFrameDataSet, wg *sync.WaitGroup) {

	thdr := rw.t.Lookup("codeframe_hdr.tmpl")
	trow := rw.t.Lookup("codeframe_row.tmpl")

	// create directory for the output
	fpath := "out/"
	err := os.MkdirAll(fpath, os.ModePerm)
	check(err)

	// create the report data file in the output directory
	// delete any ecisting files and create empty new one
	fname := fpath + "codeframe.dat"
	err = os.RemoveAll(fname)
	f, err := os.Create(fname)
	check(err)
	defer f.Close()

	// write the data - ignoring writing domain items, reported separately
	for _, cfd := range cfds {
		if !(cfd.Test.TestContent.TestDomain == "Writing") {
			if err := trow.Execute(f, cfd); err != nil {
				check(err)
			}

		}
	}

	// write the empty header file
	fname2 := fpath + "codeframe.csv"
	f2, err := os.Create(fname2)
	check(err)
	defer f2.Close()

	// doesn't actually need any data - all text fields so pass nil struct as data
	if err := thdr.Execute(f2, nil); err != nil {
		check(err)
	}

	inputFile := []string{fname}
	outputFile := fname2

	concatenateFiles(inputFile, outputFile)

	// remove the temp data files
	err = os.RemoveAll(fname)
	check(err)

	log.Printf("Codeframe report created for: %d elements", len(cfds))

	wg.Done()

}

func (rw *ReportWriter) writeDomainScoreReport(acaraid string) {

	thdr := rw.t.Lookup("domainscore_hdr.tmpl")
	trow := rw.t.Lookup("domainscore_row.tmpl")

	// create directory for the school
	fpath := "out/" + acaraid
	err := os.MkdirAll(fpath, os.ModePerm)
	check(err)

	// create the report data file in the directory
	// delete any existing files and create empty new one
	fname := fpath + "/domain_scores.dat"
	err = os.RemoveAll(fname)
	f, err := os.Create(fname)
	check(err)
	defer f.Close()

	// write the data
	rds := rw.sr.GetDomainScoreData(acaraid)
	for _, rd := range rds {
		if err := trow.Execute(f, rd); err != nil {
			check(err)
		}
	}

	// write the empty header file
	fname2 := fpath + "/domain_scores.csv"
	f2, err := os.Create(fname2)
	check(err)
	defer f2.Close()

	// doesn't actually need any data - all text fields so pass nil struct as data
	if err := thdr.Execute(f2, nil); err != nil {
		check(err)
	}

	inputFile := []string{fname}
	outputFile := fname2

	concatenateFiles(inputFile, outputFile)

	log.Printf("Domain scores report created for: %s %d response-sets", acaraid, len(rds))

}

func (rw *ReportWriter) writeParticipationReport(acaraid string) {

	thdr := rw.t.Lookup("participation_hdr.tmpl")
	trow := rw.t.Lookup("participation_row.tmpl")

	// create directory for the school
	fpath := "out/" + acaraid
	err := os.MkdirAll(fpath, os.ModePerm)
	check(err)

	// create the report data file in the directory
	// delete any ecisting files and create empty new one
	fname := fpath + "/participation.dat"
	err = os.RemoveAll(fname)
	f, err := os.Create(fname)
	check(err)
	defer f.Close()

	// write the data
	pds := rw.sr.GetParticipationData(acaraid)
	for _, pd := range pds {
		if err := trow.Execute(f, pd); err != nil {
			check(err)
		}
	}

	// write the empty header file
	fname2 := fpath + "/participation.csv"
	f2, err := os.Create(fname2)
	check(err)
	defer f2.Close()

	// doesn't actually need any data - all text fields so pass nil struct as data
	if err := thdr.Execute(f2, nil); err != nil {
		check(err)
	}

	inputFile := []string{fname}
	outputFile := fname2

	concatenateFiles(inputFile, outputFile)

	log.Printf("Participation report created for: %s %d students", acaraid, len(pds))

}

func (rw *ReportWriter) splitNAPXMLperSchool(acaraid string) {

	// create directory for the school
	fpath := "out/" + acaraid
	err := os.MkdirAll(fpath, os.ModePerm)
	check(err)
	sd := rw.sr.GetSchoolData(acaraid)

	// create the report data file in the directory
	// delete any existing files and create empty new one
	fname := fpath + "/napevents.xml"
	err = os.RemoveAll(fname)
	f, err := os.Create(fname)
	check(err)
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	f.WriteString("<NAPEventStudentLinks xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\n")
	defer f.Close()
	// write the data
	for _, val := range sd.Events {
		enc.Encode(val)
	}
	enc.Flush()
	f.WriteString("</NAPEventStudentLinks>\n")

	fname = fpath + "/students.xml"
	err = os.RemoveAll(fname)
	f, err = os.Create(fname)
	check(err)
	enc = xml.NewEncoder(f)
	enc.Indent("", "  ")
	f.WriteString("<StudentPersonals xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\n")
	defer f.Close()
	// write the data
	for _, val := range sd.Students {
		enc.Encode(val)
	}
	enc.Flush()
	f.WriteString("</StudentPersonals>\n")

	fname = fpath + "/napresponses.xml"
	err = os.RemoveAll(fname)
	f, err = os.Create(fname)
	check(err)
	enc = xml.NewEncoder(f)
	enc.Indent("", "  ")
	f.WriteString("<NAPStudentResponseSets xmlns=\"http://www.sifassociation.org/datamodel/au/3.4\">\n")
	defer f.Close()
	// write the data
	for _, val := range sd.Responses {
		enc.Encode(val)
	}
	enc.Flush()
	f.WriteString("</NAPStudentResponseSets>\n")

	log.Printf("XML split created for: %s %d students", acaraid, len(sd.Students))

}

func (rw *ReportWriter) writeScoreSummaryReport(acaraid string) {

	thdr := rw.t.Lookup("score_summary_hdr.tmpl")
	trow := rw.t.Lookup("score_summary_row.tmpl")

	// create directory for the school
	fpath := "out/" + acaraid
	err := os.MkdirAll(fpath, os.ModePerm)
	check(err)

	// create the report data file in the directory
	// delete any ecisting files and create empty new one
	fname := fpath + "/score_summary.dat"
	err = os.RemoveAll(fname)
	f, err := os.Create(fname)
	check(err)
	defer f.Close()

	// write the data
	ssds := rw.sr.GetScoreSummaryData(acaraid)
	for _, ssd := range ssds {
		if err := trow.Execute(f, ssd); err != nil {
			check(err)
		}
	}

	// write the empty header file
	fname2 := fpath + "/score_summary.csv"
	f2, err := os.Create(fname2)
	check(err)
	defer f2.Close()

	// doesn't actually need any data - all text fields so pass nil struct as data
	if err := thdr.Execute(f2, nil); err != nil {
		check(err)
	}

	inputFile := []string{fname}
	outputFile := fname2

	concatenateFiles(inputFile, outputFile)

	log.Printf("School score summary report created for: %s", acaraid)

}

// take a set of input files and create a single merged output file
func concatenateFiles(inputFiles []string, outputFile string) {

	reader, err := createReader(inputFiles)
	if err != nil {
		printAndHold(fmt.Sprintf("An error occurred during read: %s", err.Error()))
		return
	}

	writer, err := createWriter(outputFile)
	if err != nil {
		printAndHold(fmt.Sprintf("An error occurred during write: %s", err.Error()))
		return
	}

	err = pipe(reader, writer)
	if err != nil {
		printAndHold(fmt.Sprintf("An error occurred during pipe: %s", err.Error()))
	}

}

func createReader(filePaths []string) (reader io.Reader, err error) {
	readers := []io.Reader{}
	for _, filePath := range filePaths {
		inputFile, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		readers = append(readers, inputFile)
		// readers = append(readers, newLineReader())
	}

	return io.MultiReader(readers...), nil
}

func createWriter(filePath string) (writer *bufio.Writer, err error) {

	// aggregate output file must be opened as append to
	// maintain headers
	outputFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err != nil {
		return nil, err
	}

	return bufio.NewWriter(outputFile), nil
}

func pipe(reader io.Reader, writer *bufio.Writer) (err error) {
	_, err = writer.ReadFrom(reader)
	if err != nil {
		return
	}

	err = writer.Flush()
	if err != nil {
		return
	}

	return
}

func newLineReader() io.Reader {
	newLine := []byte("\r\n")
	return bytes.NewReader(newLine)
}

func printAndHold(msg string) {
	fmt.Println(msg)
	fmt.Scan()
}

func check(e error) {
	if e != nil {
		log.Println("Error writing report file: ", e)
	}
}
