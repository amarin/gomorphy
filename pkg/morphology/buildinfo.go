package morphology

import "time"

// BuildInfo — диагностические метаданные словаря (секция "info" файла
// GMOR): когда и чем собран, откуда данные. Ни одно поле не требуется
// для работы Parse/Lemma/Fuzzy — все поля опциональны.
type BuildInfo struct {
	BuiltAt        time.Time
	LibraryVersion string
	Source         string
	SourceVersion  string
	Author         string
	Description    string
	SourceURL      string
}

// Info возвращает диагностические метаданные словаря, если они есть в
// файле. nil — для словарей без секции "info": файлы, собранные до её
// появления, или словари, собранные вручную через Builder API и ни разу
// не прошедшие через SaveTo.
func (x *Dictionary) Info() *BuildInfo {
	if x == nil || x.d == nil || x.d.Info == nil {
		return nil
	}
	info := *x.d.Info
	return &BuildInfo{
		BuiltAt:        info.BuiltAt,
		LibraryVersion: info.LibraryVersion,
		Source:         info.Source,
		SourceVersion:  info.SourceVersion,
		Author:         info.Author,
		Description:    info.Description,
		SourceURL:      info.SourceURL,
	}
}
