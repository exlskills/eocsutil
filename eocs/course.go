package eocs

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/exlskills/eocsutil/config"
	"github.com/exlskills/eocsutil/eocs/esmodels"
	"github.com/exlskills/eocsutil/ir"
	"github.com/exlskills/eocsutil/mdutils"
	"github.com/exlskills/eocsutil/olx/olxproblems"
	"github.com/exlskills/eocsutil/wsenv"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/olivere/elastic"
	"github.com/remeh/sizedwaitgroup"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type parserCtx struct {
	course  *Course
	chapIdx int
	seqIdx  int
	vertIdx int
	n       int
	swg     *sizedwaitgroup.SizedWaitGroup
}

var olxProblemChoiceHintsMdRegex = regexp.MustCompile(`(?s-i)<choicehint.+?<\/choicehint>`)

func resolveCourseRecursive(rootDir string) (*Course, error) {
	Log.Infof("Root Directory %s", rootDir)
	rootCourseYAML, err := getIndexYAML(rootDir)
	if err != nil {
		return nil, err
	}
	c := &Course{}
	err = yaml.Unmarshal(rootCourseYAML, c)
	if err != nil {
		return nil, err
	}
	swgV := sizedwaitgroup.New(5)
	pcx := &parserCtx{
		course:  c,
		chapIdx: -1,
		seqIdx:  -1,
		vertIdx: -1,
		n:       0,
		swg:     &swgV,
	}
	err = filepath.Walk(rootDir, courseWalkFunc(rootDir, pcx))
	if err != nil {
		return nil, err
	}
	Log.Info("Returned from course directory scanning. Waiting for workers to return ...")
	pcx.swg.Wait()
	Log.Info("All course content workers returned.")
	err = checkWalkErrors(*c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func courseWalkFunc(rootDir string, pcx *parserCtx) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		defer func() { pcx.n++ }()
		if pcx.n == 0 {
			// The first iteration is always the parent directory, so skip it
			return nil
		}
		base := filepath.Base(path)
		ext := filepath.Ext(path)
		if err != nil {
			Log.Error(err)
			return nil
		}
		if !info.IsDir() || ext == ".repl" {
			// We always ignore these since they are handled by other import funcs
			return nil
		}
		// Now we know that we've hit a directory...
		if isIgnoredDir(base) {
			return filepath.SkipDir
		}
		pathParts := strings.Split(strings.Replace(path, rootDir+string(filepath.Separator), "", 1), string(filepath.Separator))
		if len(pathParts) == 1 {
			// Create a new chapter
			pcx.vertIdx = -1
			pcx.seqIdx = -1
			pcx.chapIdx++
			chap := &Chapter{}
			_, dispName, err := indexAndNameFromConcatenated(base)
			if err != nil {
				return err
			}
			if indxBytes, err := getIndexYAML(path); err == nil {
				err = yaml.Unmarshal(indxBytes, &chap)
				if err != nil {
					return err
				}
				dispName = chap.DisplayName
				if chap.URLName == "" {
					chap.URLName = esmodels.ESID()
					// Persist the ID
					err := writeIndexYAML(path, chap)
					if err != nil {
						return err
					}
				}
			} else {
				chap.URLName = esmodels.ESID()
				chap.DisplayName = dispName
				// Persist the ID
				err := writeIndexYAML(path, chap)
				if err != nil {
					return err
				}
			}
			chap.Index = pcx.chapIdx
			pcx.course.Chapters = append(pcx.course.Chapters, chap)
		} else if len(pathParts) == 2 {
			// Create a new sequential
			Log.Debug("Sequential ", path)
			pcx.vertIdx = -1
			pcx.seqIdx++
			seq := &Sequential{}
			_, dispName, err := indexAndNameFromConcatenated(base)
			if err != nil {
				return err
			}
			if indxBytes, err := getIndexYAML(path); err == nil {
				err = yaml.Unmarshal(indxBytes, &seq)
				if err != nil {
					return err
				}
				dispName = seq.DisplayName
				if seq.URLName == "" {
					seq.URLName = esmodels.ESID()
					// Persist the ID
					err := writeIndexYAML(path, seq)
					if err != nil {
						return err
					}
				}
			} else {
				seq.URLName = esmodels.ESID()
				seq.DisplayName = dispName
				// Persist the ID
				err := writeIndexYAML(path, seq)
				if err != nil {
					return err
				}
			}
			pcx.course.Chapters[pcx.chapIdx].Sequentials = append(pcx.course.Chapters[pcx.chapIdx].Sequentials, seq)
		} else if len(pathParts) == 3 {
			// Create an index a new vertical
			pcx.vertIdx++
			vert := &Vertical{}
			_, dispName, err := indexAndNameFromConcatenated(base)
			if err != nil {
				return err
			}
			Log.Debug("Adding vertical: ", dispName)
			if indxBytes, err := getIndexYAML(path); err == nil {
				err = yaml.Unmarshal(indxBytes, &vert)
				if err != nil {
					return err
				}
				dispName = vert.DisplayName
				if vert.URLName == "" {
					vert.URLName = esmodels.ESID()
					// Persist the ID
					err := writeIndexYAML(path, vert)
					if err != nil {
						return err
					}
				}
			} else {
				vert.URLName = esmodels.ESID()
				vert.DisplayName = dispName
				// Persist the ID
				err := writeIndexYAML(path, vert)
				if err != nil {
					return err
				}
			}
			pcx.swg.Add()
			go blockExtractionRoutine(pcx.swg, vert, path)
			for _, b := range vert.Blocks {
				Log.Debugf("After blockExtractionRoutine. Block type %s, path  %s", b.BlockType, b.FSPath)
			}
			pcx.course.Chapters[pcx.chapIdx].Sequentials[pcx.seqIdx].Verticals = append(pcx.course.Chapters[pcx.chapIdx].Sequentials[pcx.seqIdx].Verticals, vert)
			// Since the vertical directory was handled by the 'extractBlocks' func above, we want to keep moving...
			return filepath.SkipDir
		} else {
			Log.Println(pathParts)
			Log.Println(pcx)
			return errors.New("eocs: invalid directory depth/name combination")
		}
		return nil
	}
}

func blockExtractionRoutine(wg *sizedwaitgroup.SizedWaitGroup, vert *Vertical, path string) {
	defer wg.Done()
	var err error
	vert.Blocks, err = extractBlocksFromVerticalDirectory(path)
	if err != nil {
		// Log.Fatalf("Encountered fatal error processing blocks for vertical %s (ID: %s), error: %s", vert.DisplayName, vert.URLName, err.Error())
		// Need to ensure a clean program exit as well as continue validation
		// Append a dummy ERROR Block
		vert.Blocks = append(vert.Blocks, &Block{
			BlockType: "ERROR",
			Markdown:  fmt.Sprintf("%v", err),
		})
	}
}

func extractBlocksFromVerticalDirectory(rootPath string) (blks []*Block, err error) {
	rootPathParts := strings.Split(rootPath, "/")
	if len(rootPathParts) < 4 {
		return nil, errors.New("invalid path to block, must contain at least 4 directories (course->chapter->sequential->vertical) to form valid EOCS structure")
	}
	rootPathParts = rootPathParts[len(rootPathParts)-3:]
	vertDirListing, err := ioutil.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}
	// We ignore all directories here, until they become explicitly imported by a repl or other method
	for _, fi := range vertDirListing {
		if strings.HasSuffix(fi.Name(), ".prob.md") {
			// Parse as `problem` block
			byteContents, err := ioutil.ReadFile(filepath.Join(rootPath, fi.Name()))
			if err != nil {
				return nil, err
			}
			prob, err := olxproblems.NewProblemFromMD(string(byteContents))
			if err != nil {
				Log.Error("Encountered error in file: ", filepath.Join(rootPath, fi.Name()))
				return nil, err
			}
			var rpl *BlockREPL
			if prob.StringResponse != nil && strings.HasPrefix(prob.StringResponse.Answer, "#!") {
				// Start looking for the REPL
				yamlName, err := getProblemREPLPath(prob.StringResponse.Answer)
				if err != nil {
					return nil, err
				}
				rplYamlContents, err := ioutil.ReadFile(filepath.Join(rootPath, yamlName))
				if err != nil {
					return nil, err
				}
				rpl, err = loadReplForEOCS(rplYamlContents, rootPath)
				if err != nil {
					return nil, err
				}
			}
			blks = append(blks, &Block{
				BlockType: "problem",
				// NOTE: This URLName is not actually used in the EXLskills import, so it is okay to set it on each load...
				URLName:     esmodels.ESID(),
				DisplayName: strings.SplitN(fi.Name(), ".", 2)[0],
				Markdown:    string(byteContents),
				REPL:        rpl,
				FSPath:      filepath.Join(append(rootPathParts, fi.Name())...),
			})
		} else if strings.HasSuffix(fi.Name(), ".md") {
			// Parse as `html` block
			byteContents, err := ioutil.ReadFile(filepath.Join(rootPath, fi.Name()))
			if err != nil {
				return nil, err
			}
			blks = append(blks, &Block{
				BlockType: "html",
				// NOTE: This URLName is not actually used in the EXLskills import, so it is okay to set it on each load...
				URLName:     esmodels.ESID(),
				DisplayName: strings.SplitN(fi.Name(), ".", 2)[0],
				Markdown:    string(byteContents),
				FSPath:      filepath.Join(append(rootPathParts, fi.Name())...),
			})
		} else if strings.HasSuffix(fi.Name(), ".repl.yaml") && !strings.HasSuffix(fi.Name(), ".prob.repl.yaml") {
			// Parse as `exleditor` block
			byteContents, err := ioutil.ReadFile(filepath.Join(rootPath, fi.Name()))
			if err != nil {
				return nil, err
			}
			var rpl *BlockREPL
			rpl, err = loadReplForEOCS(byteContents, rootPath)
			if err != nil {
				return nil, err
			}
			blks = append(blks, &Block{
				BlockType: "exleditor",
				// NOTE: This URLName is not actually used in the EXLskills import, so it is okay to set it on each load...
				URLName:     esmodels.ESID(),
				DisplayName: strings.SplitN(fi.Name(), ".", 2)[0],
				REPL:        rpl,
				FSPath:      filepath.Join(append(rootPathParts, fi.Name())...),
			})
		}
	}
	return
}

func loadReplForEOCS(yamlBytes []byte, rootPath string) (rpl *BlockREPL, err error) {
	err = yaml.Unmarshal(yamlBytes, &rpl)
	if err != nil {
		return nil, err
	}
	if !rpl.IsAPIVersionValid() {
		return nil, errors.New(fmt.Sprintf("eocs: invalid repl api_version %v", rpl.APIVersion))
	}
	if !rpl.IsEnvironmentKeyValid() {
		return nil, errors.New("eocs: invalid repl environment_key " + rpl.EnvironmentKey)
	}
	err = rpl.LoadFilesFromFS(rootPath)
	if err != nil {
		return nil, err
	}
	return rpl, nil
}

// upsertCourseRecursive handles course load into the ES storage MongoDB and Elasticsearch targets
// It takes the course objects alog with the target storage parameters, calls convertToESCourse to generate storage-ready objects
// from the course object and manages the load
func upsertCourseRecursive(course *Course, mongoURI, dbName string, elasticsearchURI string, elasticsearchIndex string) (err error) {
	sess, err := mgo.DialWithTimeout(mongoURI, time.Duration(10*time.Second))
	if err != nil {
		Log.Error("MongoDB error", err)
		return err
	}
	esc, exams, qs, vcs, esearchdocs, err := convertToESCourse(course)
	if err != nil {
		return err
	}
	db := sess.DB(dbName)

	for _, q := range qs {
		// cInfo, err := db.C("question").UpsertId(q.ID, q)
		_, err := db.C("question").UpsertId(q.ID, q)
		if err != nil {
			Log.Errorf("MongoDB error with 'question' object: %v, and error: %s", q, err.Error())
			return err
		}
		// Log.Info("EXLskills 'question' changes: ", *cInfo)
	}

	for _, vc := range vcs {
		//cInfo, err := db.C("versioned_content").UpsertId(vc.ID, vc)
		_, err := db.C("versioned_content").UpsertId(vc.ID, vc)
		if err != nil {
			Log.Errorf("MongoDB error with 'versioned_content' object: %v, and error: %s", vc, err.Error())
			return err
		}
		// Log.Info("EXLskills 'versioned_content' changes: ", *cInfo)
	}

	for _, ex := range exams {
		cInfo, err := db.C("exam").UpsertId(ex.ID, ex)
		if err != nil {
			Log.Errorf("MongoDB error with 'exam' object: %v, and error: %s", ex, err.Error())
			return err
		}
		Log.Info("EXLskills 'exam' changes: ", *cInfo)
	}

	cInfo, err := db.C("course").UpsertId(esc.ID, esc)
	if err != nil {
		Log.Errorf("MongoDB error with 'course' object: %v, and error: %s", esc, err.Error())
		return err
	}
	Log.Info("EXLskills 'course' changes: ", *cInfo)

	if (len(elasticsearchURI) > 0) {
		u, err := url.Parse(elasticsearchURI)
		if err != nil {
			Log.Errorf("Elasticsearch URI is invalid: %v. Parsing error: %s", elasticsearchURI, err.Error())
			return err
		}

		var elasticSearchClient *elastic.Client

		if (u.Scheme == "https" && !config.Cfg().IsProductionMode()) {
			// This is used for testing HTTPS backends bypassing Certificate validation
			// Set ENV MODE=debug
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr}
			elasticSearchClient, err = elastic.NewClient(elastic.SetHttpClient(client), elastic.SetSniff(false), elastic.SetURL(elasticsearchURI))
		} else {
			// This is used in Production and for HTTP backend testing
			elasticSearchClient, err = elastic.NewClient(elastic.SetSniff(false), elastic.SetURL(elasticsearchURI))
		}
		if err != nil {
			Log.Errorf("Elasticsearch connection issue for URI: %v. Error: %s", elasticsearchURI, err.Error())
			return err
		}
		Log.Info("Elasticsearch connected ", elasticSearchClient)

		Log.Infof("Starting to load Elasticsearch documents. There are %v documents to load", len(esearchdocs))
		Log.Infof("Target Index %v", elasticsearchIndex+"_"+course.GetLanguage())
		elasticsearchDocs := 0
		for _, esd := range esearchdocs {
			elasticsearchDocs++
			Log.Debugf("Loading doc ID %v type %v title %v", esd.ID, esd.DocType, esd.Title)
			_, err = elasticSearchClient.Index().
				Index(elasticsearchIndex + "_" + course.GetLanguage()).
				Type("_doc").
				Id(esd.ID).
				BodyJson(esd).
				Refresh("false").
				Do(context.Background())
			if err != nil {
				// Handle error
				Log.Errorf("Elasticsearch index issue for URI: %v, and error: %s", elasticsearchURI, err.Error())
				return err
			}

		}
		Log.Infof("Elasticsearch: indexed %v documents", elasticsearchDocs)
	}

	return
}

func checkWalkErrors(c Course) error {
	var errorText strings.Builder
	for _, chapter := range c.Chapters {
		for _, sequential := range chapter.Sequentials {
			for _, vertical := range sequential.Verticals {
				for _, block := range vertical.Blocks {
					if block.BlockType == "ERROR" {
						errorText.WriteString(fmt.Sprintf("Error processing blocks for vertical %s (ID: %s), error: %s ", vertical.DisplayName, vertical.URLName, block.Markdown))
					}
				}
			}
		}
	}
	errorS := errorText.String()
	if len(errorS) > 0 {
		return errors.New(errorS)
	}
	return nil
}

// convertToESCourse takes the Course object as populated in preceding steps and generates objects corresponding to the ES course storage model:
// Four objects for the MongoDB collections and one object for the Elasticsearch index
func convertToESCourse(course *Course) (esc *esmodels.Course, exams []*esmodels.Exam, qs []*esmodels.Question, vc []*esmodels.VersionedContent, esearchdocs []*esmodels.ElasticsearchGenDoc, err error) {
	estMinutes, err := strconv.Atoi(course.GetExtraAttributes()["est_minutes"])
	if err != nil {
		// Note this is just a sensible default, I don't believe that est_minutes should crash a course conversion
		estMinutes = 600
	}
	weight, err := strconv.Atoi(course.GetExtraAttributes()["weight"])
	if err != nil {
		// Ensure default on error
		weight = 0
	}
	skillLevel, err := course.GetSkillLevel()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	esc = &esmodels.Course{
		ID:                 course.URLName,
		IsOrganizationOnly: false,
		Title:              esmodels.NewIntlStringWrapper(course.DisplayName, course.Language),
		Description:        esmodels.NewIntlStringWrapper(course.GetExtraAttributes()["description"], course.Language),
		Headline:           esmodels.NewIntlStringWrapper(course.GetExtraAttributes()["headline"], course.Language),
		SubscriptionLevel:  1,
		ViewCount:          0,
		EnrolledCount:      0,
		SkillLevel:         skillLevel,
		EstMinutes:         estMinutes,
		PrimaryTopic:       course.GetExtraAttributes()["primary_topic"],
		CoverURL:           course.GetCourseImage(),
		LogoURL:            course.GetCourseImage(),
		IsPublished:        true,
		InfoMD:             esmodels.NewIntlStringWrapper(course.GetExtraAttributes()["info_md"], course.Language),
		VerifiedCertCost:   30,
		OrganizationIDs:    []string{},
		Topics:             extraAttrCSVToStrSlice(course.GetExtraAttributes()["topics"]),
		RepoURL:            course.GetExtraAttributes()["repo_url"],
		Weight:             weight,
	}
	if course.GetExtraAttributes()["instructor_timekit"] != "" {
		instTK := esmodels.InstructorTimekit{}
		err = json.Unmarshal([]byte(course.GetExtraAttributes()["instructor_timekit"]), &instTK)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		esc.InstructorTimekit = &instTK
	}
	units, exams, qs, vc, esearchdocs, err := extractESFeatures(course)
	if err != nil {
		return
	}
	esc.Units = esmodels.UnitsWrapper{
		ID:    esmodels.ESID(),
		Units: units,
	}

	esearchdoc := &esmodels.ElasticsearchGenDoc{
		ID:          toGlobalId("Course", course.URLName),
		DocType:     "course",
		Title:       course.DisplayName,
		Headline:    course.GetExtraAttributes()["headline"],
		TextContent: course.GetExtraAttributes()["description"],
		CourseId:    course.URLName,
	}
	esearchdocs = append(esearchdocs, esearchdoc)
	return
}

func extractESFeatures(course *Course) (units []esmodels.Unit, exams []*esmodels.Exam, qs []*esmodels.Question, vc []*esmodels.VersionedContent, esearchdocs []*esmodels.ElasticsearchGenDoc, err error) {
	for _, chap := range course.Chapters {
		unit, uEx, uQs, uVcs, uEsearchdocs, err := extractESUnitFeatures(course.URLName, course.RepoURL, chap, len(course.Chapters), course.Language)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		units = append(units, unit)
		exams = append(exams, uEx...)
		qs = append(qs, uQs...)
		vc = append(vc, uVcs...)
		esearchdocs = append(esearchdocs, uEsearchdocs...)
	}
	return
}

func extractESUnitFeatures(courseID string, courseRepoUrl string, chap *Chapter, nChaps int, lang string) (unit esmodels.Unit, exams []*esmodels.Exam, qs []*esmodels.Question, vc []*esmodels.VersionedContent, esearchdocs []*esmodels.ElasticsearchGenDoc, err error) {
	Log.Debug("Extracting ESUnit Features for ", chap.DisplayName)
	unit.ID = chap.URLName
	unit.Title = esmodels.NewIntlStringWrapper(chap.DisplayName, lang)
	unit.Headline = esmodels.NewIntlStringWrapper("Learn "+chap.DisplayName, lang)
	unit.Index = chap.Index + 1
	unit.FinalExamWeightPct = (1 / float64(nChaps)) * 100
	unit.AttemptsAllowedPerDay = 2
	sections := make([]esmodels.Section, 0, len(chap.Sequentials))
	for idx, seq := range chap.Sequentials {
		if seq.GetIsGraded() && strings.HasPrefix(seq.Format, "Final Exam") {
			seqEx, seqQs, err := extractESExamFeatures(courseID, chap.URLName, seq, lang)
			if err != nil {
				return esmodels.Unit{}, nil, nil, nil, nil, err
			}
			qs = append(qs, seqQs...)
			exams = append(exams, seqEx)
			unit.FinalExamIDs = append(unit.FinalExamIDs, seqEx.ID)
		} else {
			sect, seqQs, seqVcs, sEsearchdocs, err := extractESSectionFeatures(courseID, courseRepoUrl, chap.URLName, idx, seq, lang)
			if err != nil {
				return esmodels.Unit{}, nil, nil, nil, nil, err
			}
			sections = append(sections, sect)
			qs = append(qs, seqQs...)
			vc = append(vc, seqVcs...)
			esearchdocs = append(esearchdocs, sEsearchdocs...)
		}
	}
	unit.Sections = esmodels.SectionsWrapper{
		Sections: sections,
	}

	esearchdoc := &esmodels.ElasticsearchGenDoc{
		ID:       toGlobalId("Unit", unit.ID),
		DocType:  "unit",
		Title:    chap.DisplayName,
		Headline: "Learn " + chap.DisplayName,
		CourseId: courseID,
		UnitId:   unit.ID,
	}
	esearchdocs = append(esearchdocs, esearchdoc)

	return
}

func extractEQQuestionFromBlock(courseID, unitID, sectID, quesID string, qBlk *Block, rpl *BlockREPL, lang string) (*esmodels.Question, error) {
	var qData interface{}
	var qType string
	var qLabel esmodels.IntlStringWrapper
	var qEstSecs int
	probMD, err := qBlk.GetContentMD()
	if err != nil {
		return nil, err
	}
	// Log.Info(probMD)
	olxProblem, err := olxproblems.NewProblemFromMD(probMD)
	if err != nil {
		return nil, err
	}
	var qHint esmodels.IntlStringWrapper
	if olxProblem.DemandHint != nil {
		qHint = esmodels.NewIntlStringWrapper(olxProblem.DemandHint.Hint, lang)
	}
	if olxProblem.MultipleChoiceResponse != nil {
		qEstSecs = 60
		qType = esmodels.ESTypeFromOLXType("multiplechoiceresponse")
		qLabel = esmodels.NewIntlStringWrapper(olxProblem.MultipleChoiceResponse.Label.InnerXML, lang)
		qData, err = olxChoicesToESQDataArr(olxProblem.MultipleChoiceResponse.ChoiceGroup.Choices, lang)
		if err != nil {
			return nil, err
		}
	} else if olxProblem.ChoiceResponse != nil {
		qEstSecs = 60
		qType = esmodels.ESTypeFromOLXType("choiceresponse")
		qLabel = esmodels.NewIntlStringWrapper(olxProblem.ChoiceResponse.Label.InnerXML, lang)
		qData, err = olxChoicesToESQDataArr(olxProblem.ChoiceResponse.CheckboxGroup.Choices, lang)
		if err != nil {
			return nil, err
		}
	} else if olxProblem.StringResponse != nil {
		qEstSecs = 60 * 5
		qType = esmodels.ESTypeFromOLXType("stringresponse")
		qLabel = esmodels.NewIntlStringWrapper(olxProblem.StringResponse.Label.InnerXML, lang)
		qData, err = olxStrRespToESQCodeData(lang, olxProblem.StringResponse.Answer, rpl)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New(fmt.Sprintf("invalid olx problem type: %s", olxProblem.XMLName.Local))
	}
	q := &esmodels.Question{
		ID:           quesID,
		Data:         qData,
		QuestionType: qType,
		QuestionText: qLabel,
		EstTimeSec:   qEstSecs,
		Hint:         qHint,
		DocRef: esmodels.DocRef{
			EmbeddedDocRef: esmodels.EmbeddedDocRefWrapper{
				EmbeddedDocRefs: []esmodels.EmbeddedDocRef{
					{
						DocID: courseID,
						Level: "course",
					},
					{
						DocID: unitID,
						Level: "unit",
					},
					{
						DocID: sectID,
						Level: "section",
					},
				},
			},
		},
		CourseItemRef: esmodels.CourseItemRef{
			CourseID:  courseID,
			UnitID:    unitID,
			SectionID: sectID,
		},
		// todo tags
		Tags:            []string{},
		Points:          1,
		ComplexityLevel: 1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	return q, nil
}

func extractESExamFeatures(courseID, unitID string, sequential *Sequential, lang string) (exam *esmodels.Exam, qs []*esmodels.Question, err error) {
	exam = &esmodels.Exam{}
	exam.UseIDETestMode = true
	exam.ID = sequential.URLName + "_exam"
	// 1ejFaqz00nJy is Sasha Varlamov
	exam.CreatorID = "1ejFaqz00nJy"
	// TODO pull PassMarkPct from sequential
	exam.PassMarkPct = 75
	for _, vert := range sequential.Verticals {
		var qBlk *Block
		if len(vert.Blocks) != 1 {
			return nil, nil, errors.New(fmt.Sprintf("final exam vertical should have exactly one block (a problem block) (Vertical ID: %s)", vert.URLName))
		}
		for _, b := range vert.Blocks {
			if b.BlockType == "problem" {
				qBlk = b
			} else {
				return nil, nil, errors.New("final exam vertical block must be of type 'problem'")
			}
		}
		q, err := extractEQQuestionFromBlock(courseID, unitID, sequential.URLName, vert.URLName, qBlk, qBlk.REPL, lang)
		if err != nil {
			return nil, nil, err
		}
		q.ExamOnly = true
		exam.EstTime += int(math.Round(float64(q.EstTimeSec) / 60))
		exam.TimeLimit += int(math.Round((float64(q.EstTimeSec) * 1.5) / 60))
		exam.QuestionIDs = append(exam.QuestionIDs, q.ID)
		exam.QuestionCount++
		qs = append(qs, q)
	}
	return exam, qs, nil
}

// olxStrRespToESQCodeData ans field represents the shebang (#!) that points us to the REPL configuration
func olxStrRespToESQCodeData(lang, ans string, rpl *BlockREPL) (cqd esmodels.CodeQuestionData, err error) {
	if !isValidProblemREPLShebang(ans) {
		Log.Errorf("Invalid problem shebang. Got %s for repl %v", ans, *rpl)
		return cqd, errors.New("stringresponse problem invalid answer shebang (#!)")
	}
	srcFilesJson, err := json.Marshal(rpl.SrcFiles)
	if err != nil {
		return cqd, err
	}
	tmplFilesJson, err := json.Marshal(rpl.TmplFiles)
	if err != nil {
		return cqd, err
	}
	testFilesJson, err := json.Marshal(rpl.TestFiles)
	if err != nil {
		return cqd, err
	}
	gradingTestsJson, err := json.Marshal(rpl.Tests)
	if err != nil {
		return cqd, err
	}
	strategy := "default"
	if rpl.GradingStrategy != "" {
		strategy = rpl.GradingStrategy
	}
	explWspcBytes, err := json.Marshal(wsenv.Workspace{Id: esmodels.ESID(), EnvironmentKey: rpl.EnvironmentKey, Files: rpl.SrcFiles})
	if err != nil {
		return cqd, err
	}
	explanation, err := generateCodeQuestionExplanationMD(rpl.Explanation, string(explWspcBytes))
	if err != nil {
		return cqd, err
	}
	return esmodels.CodeQuestionData{
		ID:              bson.NewObjectId(),
		APIVersion:      rpl.APIVersion,
		EnvironmentKey:  rpl.EnvironmentKey,
		SrcFiles:        esmodels.NewIntlStringWrapper(string(srcFilesJson), lang),
		TestFiles:       string(testFilesJson),
		TmplFiles:       esmodels.NewIntlStringWrapper(string(tmplFilesJson), lang),
		GradingStrategy: strategy,
		GradingTests:    string(gradingTestsJson),
		Explanation:     esmodels.NewIntlStringWrapper(explanation, lang),
	}, nil
}

func generateCodeQuestionExplanationMD(explStr, srcWorkspace string) (string, error) {
	buf := bytes.NewBufferString("")
	if explStr != "" {
		buf.WriteString(explStr)
		buf.WriteString("\n\n")
	}
	embeddedBlock := EXLcodeEmbeddedREPLBlock{Src: srcWorkspace}
	iframeBytes, err := embeddedBlock.IFrame()
	if err != nil {
		return "", err
	}
	buf.Write(iframeBytes)
	return buf.String(), nil
}

func olxStripHintsFromMD(md string) string {
	return olxProblemChoiceHintsMdRegex.ReplaceAllString(md, "")
}

func olxChoicesToESQDataArr(choices []olxproblems.Choice, lang string) ([]esmodels.AnswerChoice, error) {
	esc := make([]esmodels.AnswerChoice, 0, len(choices))
	for ind, c := range choices {
		txtMd, err := mdutils.MakeMD(olxStripHintsFromMD(c.InnerXML), "github")
		if err != nil {
			return nil, err
		}
		hintMd := ""
		if c.ChoiceHint != nil && len(c.ChoiceHint) > 0 {
			// TODO see how to handle multiple hints, since exlskills is only capable of one hint ("explanation")
			hintMd, err = mdutils.MakeMD(c.ChoiceHint[0].InnerXML, "github")
			if err != nil {
				return nil, err
			}
		}
		esc = append(esc, esmodels.AnswerChoice{
			ID: bson.NewObjectId(),
			// NOTE: This magic math comes from the course collection's schema where the seq is always (index+1)*10
			Sequence:    (ind + 1) * 10,
			Text:        esmodels.NewIntlStringWrapper(txtMd, lang),
			IsAnswer:    c.Correct,
			Explanation: esmodels.NewIntlStringWrapper(hintMd, lang),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}
	return esc, nil
}

// extractESSectionFeatures iterates over sequential.Verticals that represents the lowest level in the topic structure hierarchy
// Each element in sequential.Verticals contains one set of vert.Blocks comprising one Card
func extractESSectionFeatures(courseID, courseRepoUrl, unitID string, index int, sequential *Sequential, lang string) (section esmodels.Section, qs []*esmodels.Question, vc []*esmodels.VersionedContent, esearchdocs []*esmodels.ElasticsearchGenDoc, err error) {
	Log.Debug("Extracting ESSection Features for ", sequential.DisplayName)
	section.ID = sequential.URLName
	section.Index = index + 1
	section.Title = esmodels.NewIntlStringWrapper(sequential.DisplayName, lang)
	section.Headline = esmodels.NewIntlStringWrapper("Learn "+sequential.DisplayName, lang)
	for idx, vert := range sequential.Verticals {
		var contentBuf bytes.Buffer
		var ghEditUrl string
		var qBlks []*Block
		var cardText strings.Builder
		var cardCode strings.Builder
		for _, blk := range vert.Blocks {
			if blk.BlockType == "problem" {
				qBlks = append(qBlks, blk)
			} else if blk.BlockType == "exleditor" {
				var (
					srcStr  string
					tmplStr string
					testStr string
				)
				if blk.REPL.SrcFiles != nil {
					b, err := json.Marshal(wsenv.Workspace{
						Id:             esmodels.ESID(),
						Name:           blk.DisplayName,
						EnvironmentKey: blk.REPL.EnvironmentKey,
						Files:          blk.REPL.SrcFiles,
					})
					if err != nil {
						return section, nil, nil, nil, err
					}
					srcStr = string(b)
					cardCode.WriteString(blk.REPL.GetRawSrcFilesContentsString())
				}
				if blk.REPL.TmplFiles != nil {
					b, err := json.Marshal(wsenv.Workspace{
						Id:             esmodels.ESID(),
						Name:           blk.DisplayName,
						EnvironmentKey: blk.REPL.EnvironmentKey,
						Files:          blk.REPL.TmplFiles,
					})
					if err != nil {
						return section, nil, nil, nil, err
					}
					tmplStr = string(b)
				}
				if blk.REPL.TestFiles != nil {
					b, err := json.Marshal(wsenv.Workspace{
						Id:             esmodels.ESID(),
						Name:           blk.DisplayName,
						EnvironmentKey: blk.REPL.EnvironmentKey,
						Files:          blk.REPL.TestFiles,
					})
					if err != nil {
						return section, nil, nil, nil, err
					}
					testStr = string(b)
				}
				replBlock := &EXLcodeEmbeddedREPLBlock{
					Src:  srcStr,
					Test: testStr,
					Tmpl: tmplStr,
				}
				replBlkBytes, err := replBlock.IFrame()
				if err != nil {
					return section, nil, nil, nil, err
				}
				contentBuf.Write(replBlkBytes)
				contentBuf.WriteString("\n\n")
			} else if blk.BlockType == "html" {
				mdContent, err := blk.GetContentMD()
				if err != nil {
					return section, nil, nil, nil, err
				}
				contentBuf.WriteString(mdContent)
				contentBuf.WriteString("\n\n")
				cardText.WriteString(mdContent)
				if courseRepoUrl != "" {
					ghEditUrl, _ = esmodels.GenerateCardEditURL(courseRepoUrl, blk.FSPath)
				}
			} else {
				return section, nil, nil, nil, errors.New("invalid block type, must be problem, html, or exleditor for a vertical")
			}
		}
		qids := make([]string, 0, len(qBlks))
		for qIdx, q := range qBlks {
			ques, err := extractEQQuestionFromBlock(courseID, unitID, section.ID, fmt.Sprintf("%s_q_%d", vert.URLName, qIdx), q, q.REPL, lang)
			if err != nil {
				Log.Error(err)
				return section, nil, nil, nil, err
			}
			ques.DocRef.EmbeddedDocRef.EmbeddedDocRefs = append(ques.DocRef.EmbeddedDocRef.EmbeddedDocRefs, esmodels.EmbeddedDocRef{DocID: vert.URLName, Level: "card"})
			ques.CourseItemRef.CardID = vert.URLName
			qids = append(qids, ques.ID)
			qs = append(qs, ques)
		}
		verContent := &esmodels.VersionedContent{
			ID:            vert.URLName + "_vc",
			LatestVersion: 1,
			Contents: []esmodels.Content{
				{
					ID:      bson.NewObjectId(),
					Version: 1,
					Content: esmodels.NewIntlStringWrapper(contentBuf.String(), lang),
				},
			},
		}
		vc = append(vc, verContent)
		card := esmodels.Card{
			ID:          vert.URLName,
			Title:       esmodels.NewIntlStringWrapper(vert.DisplayName, lang),
			Headline:    esmodels.NewIntlStringWrapper("Learn "+vert.DisplayName, lang),
			Index:       idx + 1,
			ContentID:   vert.URLName + "_vc",
			QuestionIDs: qids,
			CourseItemRef: esmodels.CourseItemRef{
				CourseID:  courseID,
				UnitID:    unitID,
				SectionID: sequential.URLName,
				CardID:    vert.URLName,
			},
			GithubEditURL: ghEditUrl,
			// TODO tags
			Tags:      []string{},
			UpdatedAt: vert.UpdatedAt,
		}
		section.Cards.Cards = append(section.Cards.Cards, card)
		Log.Debug("Added Card ", vert.DisplayName)

		esearchdoc := &esmodels.ElasticsearchGenDoc{
			ID:          toGlobalId("Card", vert.URLName),
			DocType:     "card",
			Title:       vert.DisplayName,
			Headline:    "Learn " + vert.DisplayName,
			TextContent: cardText.String(),
			CodeContent: cardCode.String(),
			CourseId:    courseID,
			UnitId:      unitID,
			SectionId:   sequential.URLName,
			CardId:      vert.URLName,
		}
		esearchdocs = append(esearchdocs, esearchdoc)
	}

	esearchdoc := &esmodels.ElasticsearchGenDoc{
		ID:        toGlobalId("Section", section.ID),
		DocType:   "section",
		Title:     sequential.DisplayName,
		Headline:  "Learn " + sequential.DisplayName,
		CourseId:  courseID,
		UnitId:    unitID,
		SectionId: section.ID,
	}
	esearchdocs = append(esearchdocs, esearchdoc)
	return
}

func exportCourseRecursive(course ir.Course, rootDir string) (err error) {
	if _, err := os.Stat(rootDir); err == nil {
		return errors.New("eocs: specified root course export directory must not exist, in order to ensure that no contents are incidentally overwritten")
	}
	err = os.MkdirAll(rootDir, 0775)
	if err != nil {
		return err
	}
	courseEOCS := &Course{
		URLName:     course.GetURLName(),
		DisplayName: course.GetDisplayName(),
		Org:         course.GetOrgName(),
		CourseCode:  course.GetCourseCode(),
		CourseImage: course.GetCourseImage(),
		Language:    course.GetLanguage(),
	}
	err = writeIndexYAML(rootDir, courseEOCS)
	if err != nil {
		return err
	}

	err = appendIRChaptersToCourse(courseEOCS, course.GetChapters())
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	for chapIdx, chap := range courseEOCS.Chapters {
		wg.Add(1)
		go func(rd string, cIdx int, c *Chapter) {
			Log.Info("Starting to export chapter: ", c.DisplayName)
			err = exportChapterRecursive(rd, cIdx, c)
			Log.Info("Returned from chapter export: ", c.DisplayName)
			if err != nil {
				Log.Fatalf("eocs: chapter export routine encountered fatal error: %s", err.Error())
			}
			wg.Done()
		}(rootDir, chapIdx, chap)
	}
	wg.Wait()
	return nil
}

func exportChapterRecursive(rootDir string, index int, chap *Chapter) (err error) {
	dirName := filepath.Join(rootDir, concatDirName(index, chap.DisplayName))
	err = os.MkdirAll(dirName, 0775)
	if err != nil {
		return err
	}
	err = writeIndexYAML(dirName, chap)
	if err != nil {
		return
	}
	for seqIdx, seq := range chap.Sequentials {
		err = exportSequentialRecursive(dirName, seqIdx, seq)
		if err != nil {
			return
		}
	}
	return
}

func exportSequentialRecursive(rootDir string, index int, seq *Sequential) (err error) {
	dirName := filepath.Join(rootDir, concatDirName(index, seq.DisplayName))
	err = os.MkdirAll(dirName, 0775)
	if err != nil {
		return err
	}
	err = writeIndexYAML(dirName, seq)
	if err != nil {
		return
	}
	for vertIdx, vert := range seq.Verticals {
		err = exportVerticalRecursive(dirName, vertIdx, vert)
		if err != nil {
			return
		}
	}
	return
}

func exportVerticalRecursive(rootDir string, index int, vert *Vertical) (err error) {
	dirName := filepath.Join(rootDir, concatDirName(index, vert.DisplayName))
	err = os.MkdirAll(dirName, 0775)
	if err != nil {
		return err
	}
	err = writeIndexYAML(dirName, vert)
	if err != nil {
		return
	}

	for blkIdx, blk := range vert.Blocks {
		err = exportBlock(dirName, blkIdx, blk)
		if err != nil {
			return
		}
	}
	return
}

func exportBlock(rootDir string, index int, blk *Block) (err error) {
	fileName := filepath.Join(rootDir, concatDirName(index, blk.DisplayName))
	switch blk.GetBlockType() {
	case "exleditor":
		blk.MarshalREPL(rootDir, concatDirName(index, blk.DisplayName))
		return
	case "problem":
		fileName += ".prob.md"
	case "md", "html":
		fileName += ".md"
	default:
		return errors.New(fmt.Sprintf("eocs: unsupported block type: %s", blk.GetBlockType()))
	}
	contents, err := blk.GetContentMD()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, []byte(contents), 0755)
	if err != nil {
		return err
	}
	return nil
}

type Course struct {
	URLName           string                      `yaml:"url_name"`
	DisplayName       string                      `yaml:"display_name"`
	Org               string                      `yaml:"org"`
	CourseCode        string                      `yaml:"course"`
	CourseImage       string                      `yaml:"course_image"`
	Language          string                      `yaml:"language"`
	Headline          string                      `yaml:"headline"`
	Description       string                      `yaml:"description"`
	Topics            []string                    `yaml:"topics,flow"`
	PrimaryTopic      string                      `yaml:"primary_topic"`
	SkillLevel        string                      `yaml:"skill_level"`
	InfoMD            string                      `yaml:"info_md"`
	RepoURL           string                      `yaml:"repo_url"`
	Weight            int                         `yaml:"weight"`
	EstMinutes        int                         `yaml:"est_minutes"`
	InstructorTimekit *esmodels.InstructorTimekit `yaml:"instructor_timekit"`
	Chapters          []*Chapter                  `yaml:"-"`
}

func (course *Course) GetDisplayName() string {
	return course.DisplayName
}

func (course *Course) GetURLName() string {
	return course.URLName
}

func (course *Course) GetOrgName() string {
	return course.Org
}

func (course *Course) GetCourseCode() string {
	return course.CourseCode
}

func (course *Course) GetCourseImage() string {
	return course.CourseImage
}

func (course *Course) GetLanguage() string {
	return course.Language
}

func (course *Course) GetSkillLevel() (i int, err error) {
	if len(course.SkillLevel) == 0 {
		return 1, err
	} else {
		i, err := strconv.Atoi(course.SkillLevel)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("invalid skill_level value in index.yaml: %s", course.SkillLevel))
		}
		return i, err
	}
}

func (course *Course) GetExtraAttributes() map[string]string {
	extraAttrTK, _ := json.Marshal(course.InstructorTimekit)
	return map[string]string{
		"info_md":            course.InfoMD,
		"description":        course.Description,
		"headline":           course.Headline,
		"topics":             concatExtraAttrCSV(course.Topics),
		"primary_topic":      course.PrimaryTopic,
		"repo_url":           course.RepoURL,
		"instructor_timekit": string(extraAttrTK),
		"est_minutes":        strconv.Itoa(course.EstMinutes),
		"weight":             strconv.Itoa(course.Weight),
	}
}

func (course *Course) GetChapters() []ir.Chapter {
	return chaptersToIRChapters(course.Chapters)
}
