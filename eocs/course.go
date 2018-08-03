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
	swg *sizedwaitgroup.SizedWaitGroup
}

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
		swg: &swgV,
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
			var chap *Chapter
			idx, dispName, err := indexAndNameFromConcatenated(base)
			if err != nil {
				return err
			}
			if pcx.chapIdx != idx {
				return errors.New("invalid chapter directory index prefix")
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
			var seq *Sequential
			idx, dispName, err := indexAndNameFromConcatenated(base)
			if err != nil {
				return err
			}
			if pcx.seqIdx != idx {
				return errors.New("invalid sequential directory index prefix")
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
			var vert *Vertical
			idx, dispName, err := indexAndNameFromConcatenated(base)
			if err != nil {
				return err
			}
			Log.Info("Adding vertical: ", dispName)
			if pcx.vertIdx != idx {
				return errors.New("invalid vertical directory index prefix")
			}
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
			Log.Error("MongoDB error with 'question' object: %s", err.Error())
			return err
		}
		Log.Info("EXLskills 'question' changes: ", *cInfo)
	}

	for _, vc := range vcs {
		cInfo, err := db.C("versioned_content").UpsertId(vc.ID, vc)
		if err != nil {
			Log.Error("MongoDB error with 'versioned_content' object: %s", err.Error())
			return err
		}
		Log.Info("EXLskills 'versioned_content' changes: ", *cInfo)
	}

	for _, ex := range exams {
		cInfo, err := db.C("exam").UpsertId(ex.ID, ex)
		if err != nil {
			Log.Error("MongoDB error with 'exam' object: %s", err.Error())
			return err
		}
		Log.Info("EXLskills 'exam' changes: ", *cInfo)
	}

	cInfo, err := db.C("course").UpsertId(esc.ID, esc)
	if err != nil {
		Log.Error("MongoDB error with 'course' object: %s", err.Error())
		return err
	}
	Log.Info("EXLskills 'course' changes: ", *cInfo)

	return
}

func convertToESCourse(course *Course) (esc *esmodels.Course, exams []*esmodels.Exam, qs []*esmodels.Question, vc []*esmodels.VersionedContent, err error) {
	esc = &esmodels.Course{
		ID:                 course.URLName,
		IsOrganizationOnly: false,
		Title:              esmodels.NewIntlStringWrapper(course.DisplayName, course.Language),
		Description:        esmodels.NewIntlStringWrapper("TODO description", course.Language),
		Headline:           esmodels.NewIntlStringWrapper("TODO headline", course.Language),
		SubscriptionLevel:  1,
		ViewCount:          0,
		EnrolledCount:      0,
		SkillLevel:         1,
		EstMinutes:         3600,
		PrimaryTopic:       "Java",
		CoverURL:           course.GetCourseImage(),
		LogoURL:            course.GetCourseImage(),
		IsPublished:        true,
		InfoMD:             "TODO this needs to be converted into an intl string",
		VerifiedCertCost:   200,
		OrganizationIDs:    []string{},
		Topics:             []string{"java"},
	}
	units, exams, qs, vc, err := extractESFeatures(course)
	if err != nil {
		return
	}
	esc.Units = esmodels.UnitsWrapper{
		// TODO double-check that this can be reset each load... I don't believe that it's really ever used
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
	unit.Headline = esmodels.NewIntlStringWrapper("TODO unit headline", lang)
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
	olxProblem, err := olxproblems.NewProblemFromMD(probMD)
	if err != nil {
		return nil, err
	}
	var qHint esmodels.IntlStringWrapper
	if olxProblem.DemandHint != nil {
		hintMD, err := mdutils.MakeMD(olxProblem.DemandHint.Hint, "github")
		if err != nil {
			return nil, err
		}
		qHint = esmodels.NewIntlStringWrapper(hintMD, lang)
	}
	if olxProblem.MultipleChoiceResponse != nil {
		qEstSecs = 60
		qType = esmodels.ESTypeFromOLXType("multiplechoiceresponse")
		labelMd, err := mdutils.MakeMD(olxProblem.MultipleChoiceResponse.Label.InnerXML, "github")
		if err != nil {
			return nil, err
		}
		qLabel = esmodels.NewIntlStringWrapper(labelMd, lang)
		qData, err = olxChoicesToESQDataArr(olxProblem.MultipleChoiceResponse.ChoiceGroup.Choices, lang)
		if err != nil {
			return nil, err
		}
	} else if olxProblem.ChoiceResponse != nil {
		qEstSecs = 60
		qType = esmodels.ESTypeFromOLXType("choiceresponse")
		labelMd, err := mdutils.MakeMD(olxProblem.ChoiceResponse.Label.InnerXML, "github")
		if err != nil {
			return nil, err
		}
		qLabel = esmodels.NewIntlStringWrapper(labelMd, lang)
		qData, err = olxChoicesToESQDataArr(olxProblem.ChoiceResponse.CheckboxGroup.Choices, lang)
		if err != nil {
			return nil, err
		}
	} else if olxProblem.StringResponse != nil {
		qEstSecs = 60 * 5
		qType = esmodels.ESTypeFromOLXType("stringresponse")
		labelMd, err := mdutils.MakeMD(olxProblem.StringResponse.Label.InnerXML, "github")
		if err != nil {
			return nil, err
		}
		qLabel = esmodels.NewIntlStringWrapper(labelMd, lang)
		qData, err = olxStrRespToESQCodeData(olxProblem.StringResponse.Answer, rpl)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("invalid olx problem type")
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
					// TODO double-check that the GQL server is okay without having a card ref here... Technically final exam questions may not have one
				},
			},
		},
		// todo tags
		Tags:            []string{},
		Points:          1,
		ComplexityLevel: 1,
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
			return nil, nil, errors.New("final exam vertical should have exactly one block (a problem block)")
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
		exam.EstTime += int(math.Round(float64(q.EstTimeSec) / 60))
		exam.TimeLimit += int(math.Round((float64(q.EstTimeSec) * 1.5) / 60))
		exam.QuestionIDs = append(exam.QuestionIDs, q.ID)
		exam.QuestionCount++
		qs = append(qs, q)
	}
	return exam, qs, nil
}

// olxStrRespToESQCodeData ans field represents the shebang (#!) that points us to the REPL configuration
func olxStrRespToESQCodeData(ans string, rpl *BlockREPL) (cqd esmodels.CodeQuestionData, err error) {
	// TODO don't hard-code this... But for now we need to check that this question has been deliberately formatted
	if !isValidProblemREPLShebang(ans) {
		Log.Errorf("Invalid problem shebang. Got %s for repl %v", ans, *rpl)
		return cqd, errors.New("stringresponse problem invalid answer shebang (#!)")
	}
	return esmodels.CodeQuestionData{
		APIVersion:     rpl.APIVersion,
		EnvironmentKey: rpl.EnvironmentKey,
		SrcFiles:       rpl.SrcFiles,
		TestFiles:      rpl.TestFiles,
		TmplFiles:      rpl.TmplFiles,
	}, nil
}

func olxChoicesToESQDataArr(choices []olxproblems.Choice, lang string) ([]esmodels.AnswerChoice, error) {
	esc := make([]esmodels.AnswerChoice, 0, len(choices))
	for ind, c := range choices {
		txtMd, err := mdutils.MakeMD(c.InnerXML, "github")
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
	section.Headline = esmodels.NewIntlStringWrapper("TODO headline", lang)
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
						Name:           blk.DisplayName,
						EnvironmentKey: blk.REPL.EnvironmentKey,
						Files:          blk.REPL.TestFiles,
					})
					if err != nil {
						return section, nil, nil, err
					}
					testStr = string(b)
				}
				contentBuf.WriteString(fmt.Sprintf(`<div class="exlcode-embedded-repl" data-repl-src="%s" data-repl-test="%s" data-repl-tmpl="%s" width="100%%" height="500px"></div>`, srcStr, testStr, tmplStr))
				contentBuf.WriteString("\n")
			} else if blk.BlockType == "html" {
				mdContent, err := blk.GetContentMD()
				if err != nil {
					return section, nil, nil, err
				}
				contentBuf.WriteString(mdContent)
				contentBuf.WriteString("\n")
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
			Headline:    esmodels.NewIntlStringWrapper("TODO headline", lang),
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
	URLName     string     `yaml:"url_name"`
	DisplayName string     `yaml:"display_name"`
	Org         string     `yaml:"org"`
	CourseCode  string     `yaml:"course"`
	CourseImage string     `yaml:"course_image"`
	Language    string     `yaml:"language"`
	Chapters    []*Chapter `yaml:"-"`
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
	return map[string]string{}
}

func (course *Course) GetChapters() []ir.Chapter {
	return chaptersToIRChapters(course.Chapters)
}
