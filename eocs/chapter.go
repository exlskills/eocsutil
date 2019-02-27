package eocs

import (
	"errors"
	"github.com/exlskills/eocsutil/ir"
	"sync"
	"time"
)

func chaptersToIRChapters(chaps []*Chapter) []ir.Chapter {
	irChaps := make([]ir.Chapter, 0, len(chaps))
	for _, c := range chaps {
		irChaps = append(irChaps, c)
	}
	return irChaps
}

func appendIRChaptersToCourse(course *Course, chaps []ir.Chapter) (err error) {
	course.Chapters = make([]*Chapter, len(chaps))
	wg := sync.WaitGroup{}
	errsChan := make(chan error, len(chaps))
	chapsChan := make(chan *Chapter, len(chaps))
	for chapIdx, chap := range chaps {
		wg.Add(1)
		go func(idx int, c ir.Chapter) {
			defer wg.Done()
			newC := &Chapter{
				Index:       idx,
				URLName:     c.GetURLName(),
				DisplayName: c.GetDisplayName(),
			}
			err = appendIRSequentialsToChapter(newC, c.GetSequentials())
			if err != nil {
				errsChan <- err
			}
			chapsChan <- newC
		}(chapIdx, chap)
	}
	wg.Wait()
	close(errsChan)
	close(chapsChan)
	for err := range errsChan {
		return err
	}
	rxdChaps := 0
	for chap := range chapsChan {
		rxdChaps++
		course.Chapters[chap.Index] = chap
	}
	if len(chaps) != rxdChaps {
		return errors.New("eocs: fatal error occurred due to length mismatch of expected chapters and parsed chapters")
	}
	return nil
}

type Chapter struct {
	Index       int           `yaml:"-"`
	URLName     string        `yaml:"url_name"`
	DisplayName string        `yaml:"display_name"`
	Sequentials []*Sequential `yaml:"-"`
	UpdatedAt   time.Time     `yaml:"-"`
}

func (chap *Chapter) GetDisplayName() string {
	return chap.DisplayName
}

func (chap *Chapter) GetURLName() string {
	return chap.URLName
}

func (chap *Chapter) GetExtraAttributes() map[string]string {
	return map[string]string{}
}

func (chap *Chapter) GetSequentials() []ir.Sequential {
	return sequentialsToIRSequentials(chap.Sequentials)
}

func (chap *Chapter) SetUpdatedAt(updatedAt time.Time) {
	chap.UpdatedAt = updatedAt
}
