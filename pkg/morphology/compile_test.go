package morphology_test

import (
	"os"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const compileTestXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">существительное</grammeme>
  <grammeme id="nomn">им. п.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="дом">
   <l g="NOUN">
    <f t="дом">
     <g v="nomn"/>
    </f>
   </l>
  </lemma>
 </lemmata>
</dictionary>`

func TestCompileFromXML(t *testing.T) {
	d, err := morphology.CompileFromXML(strings.NewReader(compileTestXML), nil)
	require.NoError(t, err)
	require.NotNil(t, d)

	assert.Equal(t, "ru", d.Language())

	// Parse должен вернуть чтения для слова.
	readings := d.Parse("дом")
	assert.Greater(t, len(readings), 0, "дом должен быть найден")
}

func TestCompileFromXMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	xmlPath := tmpDir + "/dict.xml"
	datPath := tmpDir + "/out.dat"

	// Write test XML.
	require.NoError(t, os.WriteFile(xmlPath, []byte(compileTestXML), 0o644))

	// Compile and save.
	d, err := morphology.CompileFromXMLFile(xmlPath, nil)
	require.NoError(t, err)
	require.NotNil(t, d)

	// Verify parse works.
	readings := d.Parse("дом")
	assert.Greater(t, len(readings), 0)

	// Save and verify file created.
	require.NoError(t, d.SaveTo(datPath))
}

