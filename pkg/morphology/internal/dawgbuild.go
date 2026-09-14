package internal

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

// errDAWGBuild сообщает о невозможности разложить автомат в double-array:
// исчерпан диапазон представимых смещений/слотов.
var errDAWGBuild = errors.New("dawg: cannot place nodes in double-array")

// BuildDAWG строит минимальный DAWG (формат dawgdic: dictionary uint32[] +
// guide byte[]) по набору ключей.
//
// Алгоритм повторяет dawgdic (s-yata/dawgdic):
//  1. Ключи сортируются по возрастанию и инкрементально (Демьюк) собираются в
//     list-form троичный автомат. Регистр ключуется подписью ЦЕПОЧКИ братьев —
//     узел сливается только вместе со всем своим списком братьев. Это гарантия
//     того, что разделяемый (слитый) узел имеет ровно один вход со всеми
//     братьями в одинаковом составе, и double-array раскладку можно строить
//     DFS-ом с повторным использованием базы первого ребёнка.
//  2. Double-array раскладывается depth-first: дети узла (кроме первого,
//     если первый — слитый) получают свободные слоты base^label; для слитого
//     первого ребёнка переиспользуется ранее выбранная база (link-таблица),
//     что даёт один и тот же слот из всех родителей.
//
// Значение всех терминальных узлов равно 0: payload хранится как суффикс ключа
// после PayloadSeparator (см. SimilarItems). Порядок аргумента keys не
// сохраняется (срез сортируется на месте).
func BuildDAWG(keys []string) (*DAWG, error) {
	return buildDAWGWithPayload(keys)
}

// BuildDAWGWithValues строит минимальный DAWG (dawgdic format) по парам
// (key, value) и инкапсулирует value как payload: каждый ключ в DAWG
// превращается в key + PayloadSeparator + base64(value).
//
// Алгоритм полностью повторяет BuildDAWG, но ключи модифицируются
// перед сборкой. Возвращает (*DAWG, error).
func BuildDAWGWithValues(keys []string, values []uint32) (*DAWG, error) {
	if len(keys) != len(values) {
		return nil, fmt.Errorf("dawg: keys and values must have same length")
	}

	payloadKeys := make([]string, len(keys))
	for i, k := range keys {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, values[i])
		payloadKeys[i] = k + string([]byte{PayloadSeparator}) + base64.StdEncoding.EncodeToString(b)
	}

	// Reuse buildDAWGKeys with payload keys.
	return buildDAWGWithPayload(payloadKeys)
}

// buildDAWGWithPayload — shared code path для BuildDAWG и BuildDAWGWithValues.
func buildDAWGWithPayload(keys []string) (*DAWG, error) {
	sort.Strings(keys)

	b := newDawgBuilder()
	b.insertKeys(keys, nil)

	return b.compile()
}

// newDawgBuilder создаёт dawgBuilder с заранее выделенными буферами (общая
// точка входа для BuildDAWG* и BuildDAWGWithValuesProgress).
func newDawgBuilder() *dawgBuilder {
	b := &dawgBuilder{
		register:  make(map[string]int32, 1<<20),
		merged:    make([]bool, 1),
		path:      make([]int32, 0, 32),
		sigBuf:    make([]byte, 0, 64),
		labelsBuf: make([]byte, 0, 16),
	}
	b.root = b.newNode(0)
	b.path = append(b.path, b.root)

	return b
}

// insertKeys вставляет отсортированные keys в билдер (инкрементальное
// сравнение общего префикса с предыдущим ключом, Демьюк-слияние братьев).
// Если onInserted не nil, вызывается после каждой вставки с 0-based индексом
// только что вставленного ключа — используется для прогресс-коллбэков.
func (b *dawgBuilder) insertKeys(keys []string, onInserted func(i int)) {
	for i, k := range keys {
		common := 0
		for common < len(b.lastKey) && common < len(k) && b.lastKey[common] == k[common] {
			common++
		}
		b.closeSuffix(common)
		for j := common; j < len(k); j++ {
			b.appendByte(k[j])
		}
		b.nodes[b.path[len(b.path)-1]].leaf = true
		b.lastKey = k

		if onInserted != nil {
			onInserted(i)
		}
	}
	b.closeSuffix(0)
}

// dawgBuilder строит list-form DAWG (плоские узлы со списками братьев) и
// затем раскладывает его в double-array.
type dawgBuilder struct {
	nodes []dbNode

	root     int32
	path     []int32          // узлы текущего lastKey: path[0]=root
	lastKey  string           // последний вставленный ключ
	register map[string]int32 // подпись закрытой цепочки братьев → первый узел
	merged   []bool           // первый узел цепочки, слитой с другой цепочкой

	sigBuf    []byte
	labelsBuf []byte
}

// dbNode — узел list-form DAWG. Дети узла образуют цепочку братьев через next;
// первый ребёнок — nodes[first].
type dbNode struct {
	label byte
	first int32 // первый ребёнок (голова цепочки); 0 — нет детей
	next  int32 // следующий брат в родительской цепочке; 0 — нет
	leaf  bool  // узел терминальный (имеет значение)
}

func (b *dawgBuilder) newNode(label byte) int32 {
	id := int32(len(b.nodes))
	b.nodes = append(b.nodes, dbNode{label: label})
	b.merged = append(b.merged, false)
	return id
}

// appendByte добавляет новый узел первым ребёнком пути-родителя.
func (b *dawgBuilder) appendByte(label byte) {
	id := b.newNode(label)
	parent := b.path[len(b.path)-1]
	b.nodes[id].next = b.nodes[parent].first
	b.nodes[parent].first = id
	b.path = append(b.path, id)
}

// closeSuffix закрывает (минимизирует) узлы lastKey после общей части длиной
// common: узлы снимаются с пути снизу вверх и сливаются/регистрируются.
func (b *dawgBuilder) closeSuffix(common int) {
	for len(b.path) > common+1 {
		n := b.path[len(b.path)-1]
		b.path = b.path[:len(b.path)-1]
		b.replaceOrRegister(n)
	}
}

// replaceOrRegister закрывает узел n (первого ребёнка родителя): если цепочка
// братьев n уже зарегистрирована — перенаправляет ребро родителя на неё,
// иначе регистрирует n.
func (b *dawgBuilder) replaceOrRegister(n int32) {
	if n == b.root {
		return
	}
	sig := b.chainSig(n)
	if twin, ok := b.register[sig]; ok {
		b.nodes[b.path[len(b.path)-1]].first = twin
		b.merged[twin] = true
		return
	}
	b.register[sig] = n
}

// chainSig вычисляет подпись цепочки братьев, начиная с узла n: для каждого
// брата — метка, флаги (терминал, наличие брата) и id ребёнка. Ребёнок
// сравнивается по id, потому что дети закрываются раньше (снизу вверх) и
// после слияний одинаковые структуры имеют одинаковые id.
func (b *dawgBuilder) chainSig(n int32) string {
	b.sigBuf = b.sigBuf[:0]
	for n != 0 {
		b.sigBuf = append(b.sigBuf, b.nodes[n].label)
		f := byte(0)
		if b.nodes[n].leaf {
			f |= 1
		}
		if b.nodes[n].next != 0 {
			f |= 2
		}
		b.sigBuf = append(b.sigBuf, f)
		child := b.nodes[n].first
		b.sigBuf = append(b.sigBuf, byte(child>>24), byte(child>>16), byte(child>>8), byte(child))
		n = b.nodes[n].next
	}
	return string(b.sigBuf)
}

// compileWithProgress раскладывает минимизированный list-form DAWG в double-array
// (dictionary uint32[]) и строит guide. Вызывает progress callback.
func (b *dawgBuilder) compileWithProgress(progress func(processed, total int)) (*DAWG, error) {
	totalNodes := int32(len(b.nodes))
	return b.compileImpl(totalNodes, progress)
}

// compileWithTotal раскладывает DAWG и масштабирует прогресс на totalKeys.
// Это нужно когда progress callback ожидает тот же total, что и в фазе вставки ключей.
func (b *dawgBuilder) compileWithTotal(totalKeys int, progress func(processed, total int)) (*DAWG, error) {
	totalNodes := int32(len(b.nodes))
	// Wrap progress to convert nodes->keys scale
	wrappedProgress := func(processedNodes, _ int) {
		if progress != nil {
			// Map processed nodes to keys scale
			ratio := float64(processedNodes) / float64(totalNodes)
			processedKeys := int(ratio * float64(totalKeys))
			if processedKeys > totalKeys {
				processedKeys = totalKeys
			}
			// Pass totalKeys as second arg so main.go knows we're in compile phase
			progress(processedKeys, totalKeys)
		}
	}
	return b.compileImpl(totalNodes, wrappedProgress)
}

// compileImpl — shared implementation for compileWithProgress and compileWithTotal.
func (b *dawgBuilder) compileImpl(totalNodes int32, progress func(processed, total int)) (*DAWG, error) {
	if b.nodes[b.root].first == 0 {
		dic := []uint32{1 << 10}
		dic[0] = 1 << 10
		return NewDAWG(dic, nil), nil
	}

	p := newPlacer(b, totalNodes, progress)
	if !p.place(b.root, 0) {
		return nil, errDAWGBuild
	}

	if progress != nil {
		progress(int(p.processedNodes), 0)
	}

	guide := b.buildGuide(p.dic)
	if guide == nil {
		return nil, errDAWGBuild
	}
	return NewDAWG(p.dic, guide), nil
}

// placer holds the working state for laying out a minimized list-form DAWG
// into a double-array (dic): the allocator, the link table for merged
// (shared) first children, and progress bookkeeping. Split out from
// compileImpl so node placement can be tested independently of guide
// building.
type placer struct {
	b *dawgBuilder

	dic   []uint32
	alloc *slotAllocator
	link  map[int32]uint32

	processedNodes int32
	progressEvery  int32
	progress       func(processed, total int)
}

func newPlacer(b *dawgBuilder, totalNodes int32, progress func(processed, total int)) *placer {
	progressEvery := totalNodes / 10000
	if progressEvery < 10 {
		progressEvery = 10
	}

	return &placer{
		b:             b,
		dic:           []uint32{0},
		alloc:         newSlotAllocator(),
		link:          make(map[int32]uint32),
		progressEvery: progressEvery,
		progress:      progress,
	}
}

// place recursively lays out node n at double-array slot index, reusing a
// previously chosen base when n's first child is a merged (shared) node
// whose base is still valid at this index (Демьюк-style base reuse).
func (p *placer) place(n int32, index uint32) bool {
	node := &p.b.nodes[n]
	first := node.first

	if first != 0 && p.b.merged[first] {
		if base, ok := p.link[first]; ok && encodable(index^base) {
			setAt(&p.dic, index, unitAt(index^base, node.label, node.leaf))
			p.tick()
			return true
		}
	}

	p.b.labelsBuf = p.b.labelsBuf[:0]
	for e := first; e != 0; e = p.b.nodes[e].next {
		p.b.labelsBuf = append(p.b.labelsBuf, p.b.nodes[e].label)
	}

	base, ok := p.alloc.alloc(index, p.b.labelsBuf)
	if !ok {
		return false
	}

	setAt(&p.dic, index, unitAt(index^base, node.label, node.leaf))
	if node.leaf {
		setAt(&p.dic, base, isLeafBit)
	}
	if first != 0 && p.b.merged[first] {
		p.link[first] = base
	}

	for e := first; e != 0; e = p.b.nodes[e].next {
		if !p.place(e, base^uint32(p.b.nodes[e].label)) {
			return false
		}
	}

	p.tick()
	return true
}

func (p *placer) tick() {
	p.processedNodes++
	if p.progress != nil && p.processedNodes%p.progressEvery == 0 {
		p.progress(int(p.processedNodes), 0)
	}
}

// compile раскладывает минимизированный list-form DAWG в double-array
// (dictionary uint32[]) и строит guide.
func (b *dawgBuilder) compile() (*DAWG, error) {
	return b.compileWithProgress(nil)
}

// unitAt собирает transition-единицу узла: поле offset (rel) + метку и флаг
// терминала (наличие value-ребра).
func unitAt(rel uint32, label byte, hasLeaf bool) uint32 {
	var unit uint32
	if rel < 1<<21 {
		unit = rel<<10 | uint32(label)
	} else {
		unit = rel<<2 | extensionBit | uint32(label)
	}
	if hasLeaf {
		unit |= hasLeafBit
	}
	return unit
}

// encodable проверяет представимость поля offset в единице словаря (как в
// dawgdic DictionaryUnit::set_offset): значений < 1<<21 напрямую, бóльшие —
// только кратные 256 (расширенный формат).
func encodable(rel uint32) bool {
	if rel < 1<<21 {
		return true
	}
	return rel < 1<<29 && rel&0xFF == 0
}

// buildGuide строит guide (2 байта на узел: первый ребёнок + следующий брат)
// по уже разложенному double-array. Ребёнки перечисляются в порядке возрастания
// меток (как в dawgdic после инверсии цепочки при регистрации), поэтому порядок
// перечисления значений совпадает с опорным trie-сборщиком (testdawg).
func (b *dawgBuilder) buildGuide(dic []uint32) []byte {
	guide := make([]byte, len(dic)*2)
	fixed := make([]byte, (len(dic)+7)/8)

	var walk func(n int32, index uint32) bool
	walk = func(n int32, index uint32) bool {
		if fixed[index>>3]&(1<<(index&7)) != 0 {
			return true
		}
		fixed[index>>3] |= 1 << (index & 7)
		node := &b.nodes[n]
		if node.first == 0 {
			return true
		}
		order := make([]int32, 0, 8)
		for e := node.first; e != 0; e = b.nodes[e].next {
			order = append(order, e)
		}
		sort.Slice(order, func(i, j int) bool {
			return b.nodes[order[i]].label < b.nodes[order[j]].label
		})
		guide[index*2] = b.nodes[order[0]].label
		for k, e := range order {
			childIndex := index ^ offset(dic[index]) ^ uint32(b.nodes[e].label)
			if childIndex >= uint32(len(dic)) {
				return false
			}
			if !walk(e, childIndex) {
				return false
			}
			if k+1 < len(order) {
				guide[childIndex*2+1] = b.nodes[order[k+1]].label
			}
		}
		return true
	}

	if !walk(b.root, 0) {
		return nil
	}
	return guide
}

// setAt присваивает unit в dic, доращивая срез при необходимости.
func setAt(dic *[]uint32, index, unit uint32) {
	if uint32(len(*dic)) <= index {
		*dic = append(*dic, make([]uint32, index-uint32(len(*dic))+1)...)
	}
	(*dic)[index] = unit
}
