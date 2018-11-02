package eocs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/exlskills/eocsutil/eocs/esmodels"
	"github.com/exlskills/eocsutil/ir"
	"github.com/exlskills/eocsutil/mdutils"
	"github.com/exlskills/eocsutil/olx/olxproblems"
	"github.com/exlskills/eocsutil/wsenv"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/remeh/sizedwaitgroup"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math"
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
			Log.Info("Adding vertical: ", dispName)
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
		Log.Fatalf("Encountered fatal error processing blocks for vertical %s (ID: %s), error: %s", vert.DisplayName, vert.URLName, err.Error())
	}
}

func extractBlocksFromVerticalDirectory(rootPath string) (blks []*Block, err error) {
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
		return nil, errors.New("eocs: invalid repl api_version")
	}
	if !rpl.IsEnvironmentKeyValid() {
		return nil, errors.New("eocs: invalid repl environment_key")
	}
	err = rpl.LoadFilesFromFS(rootPath)
	if err != nil {
		return nil, err
	}
	return rpl, nil
}

func upsertCourseRecursive(course *Course, mongoURI, dbName string) (err error) {
	sess, err := mgo.DialWithTimeout(mongoURI, time.Duration(10*time.Second))
	if err != nil {
		return err
	}
	esc, exams, qs, vcs, err := convertToESCourse(course)
	if err != nil {
		return err
	}
	db := sess.DB(dbName)

	for _, q := range qs {
		cInfo, err := db.C("question").UpsertId(q.ID, q)
		if err != nil {
			Log.Errorf("MongoDB error with 'question' object: %v, and error: %s", q, err.Error())
			return err
		}
		Log.Info("EXLskills 'question' changes: ", *cInfo)
	}

	for _, vc := range vcs {
		cInfo, err := db.C("versioned_content").UpsertId(vc.ID, vc)
		if err != nil {
			Log.Errorf("MongoDB error with 'versioned_content' object: %v, and error: %s", vc, err.Error())
			return err
		}
		Log.Info("EXLskills 'versioned_content' changes: ", *cInfo)
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

	return
}

func convertToESCourse(course *Course) (esc *esmodels.Course, exams []*esmodels.Exam, qs []*esmodels.Question, vc []*esmodels.VersionedContent, err error) {
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
	esc = &esmodels.Course{
		ID:                 course.URLName,
		IsOrganizationOnly: false,
		Title:              esmodels.NewIntlStringWrapper(course.DisplayName, course.Language),
		Description:        esmodels.NewIntlStringWrapper(course.GetExtraAttributes()["description"], course.Language),
		Headline:           esmodels.NewIntlStringWrapper(course.GetExtraAttributes()["headline"], course.Language),
		SubscriptionLevel:  1,
		ViewCount:          0,
		EnrolledCount:      0,
		SkillLevel:         1,
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
			return nil, nil, nil, nil, err
		}
		esc.InstructorTimekit = &instTK
	}
	units, exams, qs, vc, err := extractESFeatures(course)
	if err != nil {
		return
	}
	esc.Units = esmodels.UnitsWrapper{
		ID:    esmodels.ESID(),
		Units: units,
	}
	return
}

func extractESFeatures(course *Course) (units []esmodels.Unit, exams []*esmodels.Exam, qs []*esmodels.Question, vc []*esmodels.VersionedContent, err error) {
	for _, chap := range course.Chapters {
		unit, uEx, uQs, uVcs, err := extractESUnitFeatures(course.URLName, chap, len(course.Chapters), course.Language)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		units = append(units, unit)
		exams = append(exams, uEx...)
		qs = append(qs, uQs...)
		vc = append(vc, uVcs...)
	}
	return
}

func extractESUnitFeatures(courseID string, chap *Chapter, nChaps int, lang string) (unit esmodels.Unit, exams []*esmodels.Exam, qs []*esmodels.Question, vc []*esmodels.VersionedContent, err error) {
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
				return esmodels.Unit{}, nil, nil, nil, err
			}
			qs = append(qs, seqQs...)
			exams = append(exams, seqEx)
			unit.FinalExamIDs = append(unit.FinalExamIDs, seqEx.ID)
		} else {
			sect, seqQs, seqVcs, err := extractESSectionFeatures(courseID, chap.URLName, idx, seq, lang)
			if err != nil {
				return esmodels.Unit{}, nil, nil, nil, err
			}
			sections = append(sections, sect)
			qs = append(qs, seqQs...)
			vc = append(vc, seqVcs...)
		}
	}
	unit.Sections = esmodels.SectionsWrapper{
		Sections: sections,
	}
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
	Log.Info(probMD)
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

func extractESSectionFeatures(courseID, unitID string, index int, sequential *Sequential, lang string) (section esmodels.Section, qs []*esmodels.Question, vc []*esmodels.VersionedContent, err error) {
	section.ID = sequential.URLName
	section.Index = index + 1
	section.Title = esmodels.NewIntlStringWrapper(sequential.DisplayName, lang)
	section.Headline = esmodels.NewIntlStringWrapper("Learn "+sequential.DisplayName, lang)
	for idx, vert := range sequential.Verticals {
		var contentBuf bytes.Buffer
		var qBlks []*Block
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
						return section, nil, nil, err
					}
					srcStr = string(b)
				}
				if blk.REPL.TmplFiles != nil {
					b, err := json.Marshal(wsenv.Workspace{
						Id:             esmodels.ESID(),
						Name:           blk.DisplayName,
						EnvironmentKey: blk.REPL.EnvironmentKey,
						Files:          blk.REPL.TmplFiles,
					})
					if err != nil {
						return section, nil, nil, err
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
						return section, nil, nil, err
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
					return section, nil, nil, err
				}
				contentBuf.Write(replBlkBytes)
				contentBuf.WriteString("\n\n")
			} else if blk.BlockType == "html" {
				mdContent, err := blk.GetContentMD()
				if err != nil {
					return section, nil, nil, err
				}
				contentBuf.WriteString(mdContent)
				contentBuf.WriteString("\n\n")
			} else {
				return section, nil, nil, errors.New("invalid block type, must be problem, html, or exleditor for a vertical")
			}
		}
		qids := make([]string, 0, len(qBlks))
		for qIdx, q := range qBlks {
			ques, err := extractEQQuestionFromBlock(courseID, unitID, section.ID, fmt.Sprintf("%s_q_%d", vert.URLName, qIdx), q, q.REPL, lang)
			if err != nil {
				Log.Error(err)
				return section, nil, nil, err
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
			CardRef: esmodels.DocRef{
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
							DocID: sequential.URLName,
							Level: "section",
						},
						{
							DocID: vert.URLName,
							Level: "card",
						},
					},
				},
			},
			CourseItemRef: esmodels.CourseItemRef{
				CourseID:  courseID,
				UnitID:    unitID,
				SectionID: sequential.URLName,
				CardID:    vert.URLName,
			},
			// TODO tags
			Tags: []string{},
		}
		section.Cards.Cards = append(section.Cards.Cards, card)
	}
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
