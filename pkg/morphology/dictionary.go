// Package morphology — публичный API морфологического анализа поверх
// словарей в едином внутреннем формате (pymorphy2, OpenCorpora, UniMorph).
package morphology

import (
	"github.com/amarin/gomorphy/internal/mmapx"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Dictionary — иммутабельный словарь, загруженный импортёром или открытый
// из файла GMOR (Open). Словари, открытые через Open, используют mmap:
// их необходимо закрывать методом Close, когда они больше не нужны.
type Dictionary struct {
	d  *internal.Dictionary
	mm *mmapx.Region
}
