package internal

// slotAllocator подбирает свободные base-слоты double-array раскладки
// методом intrusive doubly-linked free list (техника dawgdic/cedar/Darts,
// Aoe 1989): свободные слоты связаны в список через next/prev, поиск идёт
// только по нему — уже занятые слоты никогда не пересматриваются заново,
// в отличие от линейного сканирования битсета с нуля на каждый узел.
//
// Слот 0 зарезервирован под корень DAWG (compileImpl всегда размещает
// корень по индексу 0) и служит sentinel-значением "нет соседа" для
// списка свободных: он никогда не свободен, поэтому 0 однозначно значит
// "конец/начало списка".
type slotAllocator struct {
	used []bool
	next []uint32 // next[i]: следующий свободный слот после i; 0 — конца списка нет
	prev []uint32 // prev[i]: предыдущий свободный слот перед i; 0 — начала списка нет
	head uint32   // первый свободный слот; 0, если список пуст
	tail uint32   // последний свободный слот; 0, если список пуст
	hint [256]uint32
}

// maxSlotAllocatorCap — защитный потолок: реальные словари никогда его не
// достигают (сотни миллионов узлов), это лишь замена прежнему
// errDAWGBuild-пути на случай патологического набора ключей.
const maxSlotAllocatorCap = 1 << 30

func newSlotAllocator() *slotAllocator {
	return &slotAllocator{
		used: []bool{true}, // слот 0 занят с самого начала (корень)
		next: []uint32{0},
		prev: []uint32{0},
	}
}

func (a *slotAllocator) cap() uint32 { return uint32(len(a.used)) }

// grow расширяет ёмкость минимум до n слотов, добавляя новые слоты в
// хвост списка свободных по возрастанию индекса.
func (a *slotAllocator) grow(n uint32) {
	old := a.cap()
	if n <= old {
		return
	}
	a.used = append(a.used, make([]bool, n-old)...)
	a.next = append(a.next, make([]uint32, n-old)...)
	a.prev = append(a.prev, make([]uint32, n-old)...)
	for i := old; i < n; i++ {
		a.linkTail(i)
	}
}

func (a *slotAllocator) linkTail(i uint32) {
	if a.head == 0 {
		a.head = i
		a.tail = i
		return
	}
	a.next[a.tail] = i
	a.prev[i] = a.tail
	a.tail = i
}

func (a *slotAllocator) unlink(i uint32) {
	p, n := a.prev[i], a.next[i]
	if p != 0 {
		a.next[p] = n
	} else {
		a.head = n
	}
	if n != 0 {
		a.prev[n] = p
	} else {
		a.tail = p
	}
	a.prev[i], a.next[i] = 0, 0
}

// markUsed резервирует слот i, при необходимости расширяя ёмкость.
// No-op, если слот уже занят.
func (a *slotAllocator) markUsed(i uint32) {
	if i >= a.cap() {
		a.grow(i + 1)
	}
	if a.used[i] {
		return
	}
	a.used[i] = true
	a.unlink(i)
}

// isUsed сообщает, занят ли слот i (слоты за пределами текущей ёмкости
// считаются свободными — они ещё не были никому нужны).
func (a *slotAllocator) isUsed(i uint32) bool {
	return i < a.cap() && a.used[i]
}

// fits проверяет, что base сам свободен и все base^label для labels тоже
// свободны — т.е. узел можно разместить с этим base без коллизий.
func (a *slotAllocator) fits(base uint32, labels []byte) bool {
	if base == 0 || a.isUsed(base) {
		return false
	}
	for _, l := range labels {
		if a.isUsed(base ^ uint32(l)) {
			return false
		}
	}
	return true
}

// commit резервирует base и все дочерние слоты base^label.
func (a *slotAllocator) commit(base uint32, labels []byte) {
	a.markUsed(base)
	for _, l := range labels {
		a.markUsed(base ^ uint32(l))
	}
}

// alloc находит и резервирует base для узла на позиции index: сам base и
// все base^label для labels должны быть свободны, а index^base —
// представимо в offset-поле единицы словаря (encodable). Поиск идёт по
// списку свободных слотов начиная с подсказки для первого лейбла (если
// есть и всё ещё свободна), иначе с начала списка. При исчерпании списка
// без успеха — расширяет ёмкость вдвое и делает полный проход заново;
// такое случается только на границах роста, поэтому суммарная стоимость
// повторных полных проходов ограничена O(n log n), а не O(n) на узел.
func (a *slotAllocator) alloc(index uint32, labels []byte) (uint32, bool) {
	var startLabel byte
	if len(labels) > 0 {
		startLabel = labels[0]
	}

	base := a.head
	if h := a.hint[startLabel]; h != 0 && !a.isUsed(h) {
		base = h
	}

	for {
		for ; base != 0; base = a.next[base] {
			if encodable(index^base) && a.fits(base, labels) {
				a.commit(base, labels)
				a.hint[startLabel] = base
				return base, true
			}
		}
		if a.cap() >= maxSlotAllocatorCap {
			return 0, false
		}
		newCap := a.cap() * 2
		if newCap > maxSlotAllocatorCap {
			newCap = maxSlotAllocatorCap
		}
		a.grow(newCap)
		base = a.head
	}
}
