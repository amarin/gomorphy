# UniMorph as a dictionary data source for gomorphy

An analysis of https://unimorph.github.io/ (the home page, the Schema
section, the Sylak-Glassman (2016) PDF spec) and the actual
`unimorph/rus` dataset for Russian: assessing how applicable the
UniMorph format is as source data for the project's dictionaries.

---

## 1. What UniMorph is

**Universal Morphology (UniMorph)** is a collaborative project
annotating the world's languages' morphology in a universal schema.
The idea: any wordform of any language is represented as a pair

```
lemma  +  a set of morphological features (a bundle) from the UniMorph Schema
```

For example, the Spanish *hablaste* -> `hablar` + `FIN;IND;PFV;PST;2;SG;INFM`.
Features are given in language-independent terms, so word
representations across different languages are directly comparable
(it's an "interlingua" for inflectional morphology, in the
paradigmatic, word-based tradition).

Key facts from the site:

- As of now, **169 languages** are annotated (by `ISO 639-3` codes).
  For each language, the site shows the number of forms and paradigms,
  covered parts of speech (Nouns / Verbs / Adjectives), typology
  (agglutinative/fusional/...), data source, type
  (living/historical/...), and compatibility with shared-task splits.
- Each language's data is a separate GitHub repository,
  `github.com/unimorph/<iso>`.
- The data license is mostly **CC-BY-SA 3.0** (some languages use
  others — LGPLLR for Central Kurdish, Surrey licenses for tonal
  languages, etc.); given in the repository's README and on the site.
- There's an official Python package, `pip install unimorph` (PyPI: `unimorph`).
- The project is tied to the SIGMORPHON shared tasks (2016-2022):
  inflection, reinflection, morphological analysis.

---

## 2. The UniMorph Schema

### 2.1. Overall structure

The schema (Sylak-Glassman 2016, "The Composition and Use of the
Universal Morphological Feature Schema (UniMorph Schema)," v2 draft)
has **23 dimensions of meaning and over 212 features**.

A dimension is a general-purpose morphological category (person,
number, tense, case...). A feature is the smallest distinguishable
value within a dimension. The number of features per dimension ranges
from 2 (finiteness) to 39 (case).

Dimensions (per the document's outline; appendices 1-2 also list
Argument Marking, Possession, and Language-Specific Features):

```
Aktionsart       Finiteness               Person
Animacy          Gender / Noun Class      Polarity
Argument Marking Information Structure   Politeness
Aspect           Interrogativity          Possession
Case             Mood                     Switch-Reference
Comparison       Number                   Tense
Definiteness     Part of Speech           Valency
Deixis                                    Voice
Evidentiality    Language-Specific (*)
```

### 2.2. Example features (used in the Russian dataset)

| Dimension | Features (label) |
|---|---|
| Part of Speech | `N` (noun), `V` (verb), `ADJ`, `ADV`, `PRO`, `NUM`, `V.CVB` (converb), `V.PTCP` (participle), ... |
| Case | `NOM`, `ACC`, `GEN`, `DAT`, `INS`, `ESS` (essive/locative), `VOC`, ... |
| Number | `SG`, `PL`, `DU`, `PAUC`, ... |
| Gender | `MASC`, `FEM`, `NEUT`, `ANIM`/`INAN` (animacy), ... |
| Tense | `PRS`, `PST`, `FUT`, ... |
| Aspect | `PFV`, `IPFV`, `PRF`, ... |
| Mood | `IND`, `SBJV`, `IMP`, ... |
| Finiteness | `FIN`, `NFIN` |
| Voice | `ACT`, `PASS`, `MID`, ... |
| Language-Specific | `LGSPEC1`, `LGSPEC2`, ... |

The full registry of dimensions and features is in appendices 1 and 2
of the spec (`unimorph-schema.pdf`, pp. 60-71).

### 2.3. Bundle formation rules

- A bundle is a list of features separated by `;` (e.g. `N;ACC;SG`).
- Usually a word carries features from only a handful of dimensions;
  each dimension contributes one simple feature.
- Complex dimension values (when a simple feature isn't enough):
  - conjunction: `X+Y` (e.g. inessive = `in+ess`),
  - disjunction: `{X/Y}` (e.g. `{nom/acc}` — "nominative or accusative"),
  - negation: `non{X}` (e.g. `non{nom}` — oblique),
  - "any value of the dimension": an asterisk `*`.
  - In practice, the data published in the repositories only uses
    simple features; conjunctions/disjunctions don't occur in current datasets.
- Features are semantic: they encode meaning, not a morpheme's form.
  UniMorph doesn't store morpheme segmentation at all.

### 2.4. Relationship to Universal Dependencies

The UniMorph schema is similar in spirit to UD (v2)'s `FEATS`, but has
its own feature set. The site links to a Universal Dependencies ->
UniMorph converter (the Software section). For gomorphy, this means
the reverse is also possible — mapping UniMorph tags onto the
project's familiar (OpenCorpora-compatible) grammeme system (see §5.4).

---

## 3. Data format

Each language's data lives in a GitHub repository,
`github.com/unimorph/<iso>`, in a file named after the language code
(e.g. `rus`).

Format — **TSV, 3 columns, no header**:

```
lemma <TAB> wordform <TAB> bundle
```

- `bundle` — features separated by `;`, the feature order is preserved
  and considered meaningful.
- String encoding — UTF-8.
- The file is already sorted by lemma; each lemma's forms are
  contiguous (but sorting by lemma isn't part of the contract — an
  importer must not rely on it).
- Within one lemma, the same wordform text can repeat with
  **different** bundles — this is syncretism/homonymy (analogous to
  several ancodes for one text in OpenCorpora). There are no exact
  duplicate lines (lemma, form, bundle) in the current Russian file.
- A wordform can coincide with the lemma (the headword form) — ~43K
  such lines in the Russian file.
- Multi-word/hyphenated tokens are technically possible, e.g. `ааронов
  жезл` -> `ааронова жезла`. A space/hyphen inside the text is allowed.

An example from `unimorph/rus`:

```
ааронов жезл	ааронов жезл	N;NOM;SG
ааронов жезл	ааронова жезла	N;GEN;SG
ааронов жезл	ааронову жезлу	N;DAT;SG
ааронов	ааронова	ADJ;INAN;ACC;MASC;SG
```

The repository's README is minimal: the language name, ISO code,
source (usually Wikipedia), license.

---

## 4. The Russian dataset (measured on `unimorph/rus`)

| Metric | Value |
|---|---|
| Lines (lemma, form, bundle) | 473,482 |
| Unique lemmas | 28,069 |
| Paradigms (per the site) | 28,068 |
| Unique wordforms | 353,004 |
| Wordforms matching the lemma | 43,406 |
| Columns per line | always 3 |
| Exact duplicate lines | 0 |
| Max features in a bundle | 5 (e.g. `ADJ;INAN;ACC;MASC;SG`) |
| Part-of-speech coverage | Nouns v, Verbs v, Adjectives v |
| Typology | fusional, templatic: false |
| Source | Wikipedia |
| License | CC-BY-SA 3.0 |
| Shared-task splits | 2016 v, 2017 v |

Features used in the Russian file:

```
1 2 3  ACC ACT ADJ ANIM DAT ESS FEM FUT GEN IMP INAN INS LGSPEC1
MASC N  NEUT NFIN NOM PASS PL PRS PST SG V V.CVB V.PTCP
```

Peculiarities of the Russian annotation relevant for interpreting the tags:

- `ESS` corresponds to the Russian prepositional (locative) case.
- The `ANIM`/`INAN` marker appears in the accusative (`N;ACC;SG` vs.
  `N;ANIM;ACC;SG`), as expected for Russian's genitive-like vs.
  accusative animate distinction.
- Verbs: `NFIN` (infinitive), `V;FIN;...` with tense/person/gender,
  past-tense forms with no person (`V;PST;SG;MASC`), participles
  `V.PTCP;ACT;PST`, gerunds/converbs `V.CVB;PST`.
- `IMP` — imperative mood, `PASS` — passive voice.
- `LGSPEC1` — a language-specific feature, left uninterpreted.

The scale is significantly smaller than the OpenCorpora dictionary
(473K lines vs. ~5.5M attributed lines, 28K lemmas vs. ~392K):
UniMorph is an annotated analysis corpus based on Wikipedia, not a
full lexicon.

---

## 5. Supporting the UniMorph format in gomorphy

### 5.1. Conclusion: the format is compatible, support can be a small importer

A UniMorph line (lemma, wordform, bundle) maps **one-to-one** onto the
project's public Builder API (`pkg/dictionary`):

- `AddLemma(lemma, grammemes...)` — the lemma with the headword form's tags;
- `AddForm(lemmaID, wordform, grammemes...)` — every other line, where
  grammemes = the bundle's features.

Everything else — gomorphy's existing machinery — works unchanged from
there: paradigms (stem = LCP of the forms + suffixes), the ancode pool
(bundle -> ancode), exact wordform lookup, lemmatization, fuzzy
search, serialization into the unified `.dat`.

### 5.2. The importer pipeline

A package `pkg/morphology/importers/unimorph/` is proposed (or, in the
current structure, `pkg/unimorph/` alongside `pkg/opencorpora`):

1. **Reading.** Streaming TSV reads (`bufio.Scanner`), splitting a line
   on `\t` into exactly 3 fields.
2. **The lemma.** For a non-empty lemma — `AddLemma`. If the lemma
   column is empty (allowed by the format), treat the wordform itself
   as the lemma.
3. **Forms.** Each line -> `AddForm(lemmaID, form, features...)`, the
   features coming from `strings.Split(bundle, ";")`.
4. **Tags.** By default, UniMorph features are stored as-is (opaque
   strings, FT12); a TagSet mapping can optionally be applied (see §5.4).
5. **Compile/save.** `Compile()` -> `SaveTo()` — the same unified
   dictionary format as for OpenCorpora and PyMorphy2.

CLI surface (in the spirit of Stage 18):

```bash
gomorphy import unimorph <rus.tsv> -o ru-unimorph.dat
gomorphy -dict ru-unimorph.dat lookup кота
```

### 5.3. Complications and edge cases

| Case | Solution / note |
|---|---|
| Syncretism: one text with different bundles | Supported natively — several wordform readings (as in OpenCorpora) |
| Wordform == lemma | Attach it to the lemma as a form with its bundle; there's no separate "headword form" marker in the data |
| Empty lemma | Lemma := wordform |
| Multi-word / hyphenated tokens | The library works with arbitrary bytes; CLI lookups for such tokens need quoting |
| Feature order in a bundle is meaningful | Keep it as-is (gomorphy treats ordered ancodes as distinct, `ensureAncode`) |
| `LGSPEC*`, unknown features | Stored as opaque grammemes, no failure |
| An identical (form, bundle) within a lemma | Deduplicated by the Builder itself (pairs are interned) |
| A duplicate `(lemma, form, bundle)` | None in the current `rus`; if one appears, dedup on input |

### 5.4. Mapping UniMorph -> OpenCorpora tags (FT12)

UniMorph's and OpenCorpora's tag sets differ. TagSet normalization allows either

- **keeping UniMorph tags as-is** (the project stores tags as opaque
  strings — this is valid and gives comparability across UniMorph
  languages), or
- **projecting them onto OpenCorpora tags** for consistency with the
  main dictionary. Examples: `N→NOUN`, `V→VERB`, `ADJ→ADJF`, `ADV→ADVB`,
  `PRO→NPRO`, `NUM→NUMR`, `NOM/ACC/GEN/DAT/INS/VOC→nomn/accs/gent/datv/ablt/voct`,
  `ESS→loct`, `SG/PL→sing/plur`, `MASC/FEM/NEUT→masc/femn/neut`,
  `PRS/PST/FUT→pres/past/futr`, `PFV/IPFV→perf/impf`, `NFIN→infn`
  (imprecise: `infn` is only the infinitive, `NFIN` is any non-finite
  form; to be refined during implementation).

The projection isn't complete: features with no counterpart (e.g.
`LGSPEC1`) are either ignored during mapping or kept canonically under
their own name.

### 5.5. Impact on metrics and constraints

- By volume, the Russian UniMorph dictionary is compact (473K lines)
  and takes up a few MB in gomorphy's format.
- Lexeme coverage is several times smaller than OpenCorpora's —
  UniMorph should be used as a supplementary/reference source, not a replacement.
- The CC-BY-SA 3.0 data license imposes obligations when distributing
  derived (compiled) dictionaries — note this in the license.
- The dictionary's internal format doesn't depend on the source
  (FT8/FT11): supporting UniMorph doesn't require any on-disk format changes.

---

## 6. Recommendation

**UniMorph support can and should be implemented as a separate
importer** (`gomorphy import unimorph`, a package + tests), reusing
the Builder, paradigm compilation, and the unified `.dat` format. The
cost is minimal — it's effectively a second "lightweight" source
(after OpenCorpora), with the increment:

- `import.go` — streaming TSV -> Builder;
- unit test: 5-10 lines -> correct lemmas/paradigms;
- integration test: the entire `rus` -> `Lookup`/`Lemmas` on sample words;
- roundtrip: `import -> SaveTo -> Open -> Lookup` identical to a direct build;
- CLI: an `import unimorph` command.

Bonus: thanks to the language-independent schema, the same importer
works for all 169 UniMorph languages with no changes — this matches
requirement FT10 (language independence) and gives the project
multilingual support out of the box. It's worth adding an option to
map tags onto the OpenCorpora set to compare results against the main
Russian dictionary.
