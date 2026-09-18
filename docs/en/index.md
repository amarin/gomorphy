# gomorphy Documentation

> Russian version (installation/CLI/library basics only): [docs/ru/index.md](../ru/index.md).

## Requirements

- [Library requirements](requirements.md)
- [Roadmap](todo.md)

## Implementation

- [Implementation (as-is + plan)](implementation.md)
- [Glossary](glossary.md)

## Data source analysis

- [UniMorph: schema and data format](unimorph.md)

## Stages of the original implementation (done)

- [Stage 0. Repository analysis and preparation](implementation/stage-0-analysis.md)
- [Stage 1. Format primitives](implementation/stage-1-format.md)
- [Stage 2. String interning](implementation/stage-2-intern.md)
- [Stage 3. dict.xml scanner](implementation/stage-3-xmlscan.md)
- [Stage 4. Builder and CSR structures](implementation/stage-4-builder-csr.md)
- [Stage 5. Compiler and file loader](implementation/stage-5-compiler-loader.md)
- [Stage 6. Public facade pkg/dictionary](implementation/stage-6-facade.md)
- [Stage 7. End-to-end OpenCorpora integration](implementation/stage-7-opencorpora.md)
- [Stage 8. FT5 lemma lookup](implementation/stage-8-lemmas.md)
- [Stage 9. FT6 fuzzy search](implementation/stage-9-fuzzy.md)
- [Stage 10. Finalization](implementation/stage-10-finalize.md)

## Stages of the new implementation (paradigm + DAWG; 11-15 done, 17 partial, 16/18 planned)

- [Storage redesign rationale](implementation/redesign-rationale.md)
- [Stage 11. Internal format: TagSet + Paradigm + DAWG reader](implementation/stage-11-internal-format.md)
- [Stage 12. PyMorphy2 import](implementation/stage-12-import-pymorphy2.md)
- [Stage 13. Public API: Parse, Lemma, Fuzzy](implementation/stage-13-public-api.md)
- [Stage 14. Serialization: unified on-disk format](implementation/stage-14-serialization.md)
- [Stage 15. OpenCorpora import](implementation/stage-15-import-opencorpora.md)
- [Stage 16. UniMorph import (planned)](implementation/stage-16-import-unimorph.md)
- [DAWG build speedup (free-list)](implementation/dawg-freelist-optimization.md)
- [DAWG minimization fix (chainSig bug)](implementation/dawg-minimization-fix.md)
- [Stage 17. Narrowing ID types + format groundwork for compression](implementation/stage-17-optimize.md)
- [info section: dictionary build metadata](implementation/info-section.md)
- [Pre-1.0.0 code review: findings triage + both critical bugs](implementation/code-review-pre-1.0-triage.md)
- [Multi-dict: `morphology.MultiDictionary`](implementation/multi-dict.md)
- [Dense 1-byte DAWG alphabet for pymorphy2](implementation/pymorphy2-dense-alphabet.md)
- [pymorphy2 source (`pkg/pymorphy`) and integration into the `gomorphy` CLI](implementation/pymorphy-source-and-cli.md)
- [Universal tag mapping between dictionaries (`pkg/morphology/tagmap`)](implementation/tag-mapping.md)
- [Stage 18. Finalization: documentation, tests (planned)](implementation/stage-18-finalize.md)
- [Stage 19. Topical dictionaries: TSV import, CLI batches, skills, MCP decision](todo.md)
- [Stage 20. Synonym database: groups, tags, sidecar file](todo.md)

## Usage

- [CLI: gomorphy](cli.md)
- [Programmatic library usage](library.md)
- [Installation](installation.md)

## Review and findings

- [Pre-1.0.0 code review (full report)](code-review-pre-1.0.md)

## Research

- [DAWG packing density with a dense tag alphabet](research/0001-dawg-alphabet-density.md)
- [Binary encoding of paradigms and the tag list instead of text](research/0002-paradigm-tagset-binary-encoding.md)
- [Comparative paradigms were not merging (Cmp2/"по-")](research/0003-comparative-paradigms-not-merging.md)
- [Dense DAWG alphabet with payload](research/0004-dawg-dense-alphabet-with-payload.md)
- [Cost of a full traversal of pymorphy2's words.dawg](research/0005-pymorphy2-full-dawg-walk-cost.md)
- [UniMorph import plan](research/0006-unimorph-import-plan.md)
- [Universal Dependencies import plan](research/0007-universal-dependencies-import-plan.md)
- [Feasibility assessment for dictionary export (pymorphy2/OpenCorpora)](research/0008-dictionary-export-feasibility.md)

## Decisions

- [Why there is no built-in MCP server](mcp.md)
