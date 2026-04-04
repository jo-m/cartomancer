// Package avatar generates deterministic cartomancer/wizard-themed SVG avatars
// from a seed string. Each avatar is a 20x20 pixel SVG composed of randomized
// facial features, hair, hats, facial hair, and magical accessories.
package avatar

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// rng provides deterministic pseudo-random choices from a SHA-256 hash of the seed.
type rng struct {
	data []byte
	pos  int
}

// newRNG creates a new deterministic RNG from a seed string.
func newRNG(seed string) *rng {
	h := sha256.Sum256([]byte(seed))
	return &rng{data: h[:], pos: 0}
}

// next returns a deterministic integer in [0, n).
// When the initial hash bytes are exhausted, it re-hashes to produce more.
func (r *rng) next(n int) int {
	if n <= 0 {
		return 0
	}
	if r.pos >= len(r.data) {
		h := sha256.Sum256(r.data)
		r.data = h[:]
		r.pos = 0
	}
	v := int(r.data[r.pos]) % n
	r.pos++
	return v
}

var bgColors = [...]string{
	"#f0eae0", // warm cream
	"#e8eaf0", // cool lavender
	"#e0f0e8", // soft mint
	"#f0e8e0", // peachy white
	"#e8e8f0", // pale periwinkle
	"#f0f0e0", // light ivory
}

var skinColors = [...]string{
	"#f5dcc0", // very light
	"#f0c8a0", // light
	"#e0b080", // medium light
	"#c89060", // medium
	"#a07048", // medium dark
	"#805030", // dark
}

var hairColors = [...]string{
	"#2c1810", // near black
	"#5a3a20", // dark brown
	"#8b6040", // medium brown
	"#b07830", // auburn
	"#c0c0c0", // silver
	"#e8dcc8", // platinum
}

var robeColors = [...]string{
	"#4a2060", // deep purple
	"#2c3870", // deep blue
	"#6b3a2a", // warm brown
	"#2a4a38", // forest green
	"#6a2028", // crimson
	"#3a3050", // dark indigo
}

const (
	gold = "#c8a020"
	dark = "#2c1810"
)

const (
	numFaceShapes       = 4
	numEyeStyles        = 5
	numMouthStyles      = 5
	numHairStyles       = 4
	numFacialHairCats   = 5
	numFacialHairStyles = 3
	numHatStyles        = 5
	numClothingStyles   = 5
	numGadgets          = 7
	hatHood             = 1
	gadgetStaff         = 1
	gadgetGlasses       = 5
	gadgetNone          = 0
	gadgetTarotCard     = 2
	gadgetCrystal       = 3
	gadgetStar          = 4
	gadgetPendant       = 6
	hatWizard           = 0
	hatWideBrim         = 2
	hatCirclet          = 3
	hatTurban           = 4
)

// MakeAvatar generates a deterministic 20x20 SVG avatar depicting a cartomancer
// or wizard character. The same seed always produces the same avatar.
func MakeAvatar(seed string) string {
	r := newRNG(seed)

	bg := bgColors[r.next(len(bgColors))]
	skin := skinColors[r.next(len(skinColors))]
	hair := hairColors[r.next(len(hairColors))]
	robe := robeColors[r.next(len(robeColors))]
	faceShape := r.next(numFaceShapes)
	eyeStyle := r.next(numEyeStyles)
	mouthStyle := r.next(numMouthStyles)
	hairStyle := r.next(numHairStyles)
	facialHairCat := r.next(numFacialHairCats)
	facialHairStyle := r.next(numFacialHairStyles)
	hatStyle := r.next(numHatStyles)
	clothingStyle := r.next(numClothingStyles)
	gadget := r.next(numGadgets)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><rect width="20" height="20" fill="%s"/>`, bg)

	// Staff renders behind the character.
	if gadget == gadgetStaff {
		writeStaff(&b)
	}

	// Body renders behind the head.
	writeBody(&b, clothingStyle, skin, robe)

	// Hood background renders behind the face.
	if hatStyle == hatHood {
		fmt.Fprintf(&b, `<ellipse cx="10" cy="9" rx="6.5" ry="7" fill="%s"/>`, robe)
	}

	writeHair(&b, hairStyle, hair, hatStyle)
	writeFace(&b, faceShape, skin)
	writeEyes(&b, eyeStyle)
	writeFacialHair(&b, facialHairCat, facialHairStyle, hair)
	writeMouth(&b, mouthStyle)
	writeHat(&b, hatStyle, robe)
	writeGadget(&b, gadget, robe)

	if gadget == gadgetGlasses {
		writeGlasses(&b)
	}

	b.WriteString(`</svg>`)
	return b.String()
}

// writeBody draws the neck and shoulders at the bottom of the avatar.
func writeBody(b *strings.Builder, style int, skin, robe string) {
	// Neck.
	fmt.Fprintf(b, `<rect x="8.5" y="15" width="3" height="3" fill="%s"/>`, skin)
	// Shoulders and clothing.
	switch style {
	case 0: // Simple robe with round neckline.
		fmt.Fprintf(b, `<path d="M0,20 Q0,16 5,15.5 L8.5,17 L11.5,17 L15,15.5 Q20,16 20,20Z" fill="%s"/>`, robe)
	case 1: // V-neck tunic.
		fmt.Fprintf(b, `<path d="M0,20 Q0,16 5,15.5 L10,18.5 L15,15.5 Q20,16 20,20Z" fill="%s"/>`, robe)
	case 2: // High collar cloak.
		fmt.Fprintf(b, `<path d="M0,20 Q0,16 4,14.5 L7,15 L13,15 L16,14.5 Q20,16 20,20Z" fill="%s"/>`, robe)
		fmt.Fprintf(b, `<rect x="7" y="14.5" width="1" height="2.5" fill="%s"/>`, robe)
		fmt.Fprintf(b, `<rect x="12" y="14.5" width="1" height="2.5" fill="%s"/>`, robe)
	case 3: // Buttoned vest with shirt.
		fmt.Fprintf(b, `<path d="M0,20 Q0,16 5,15.5 L8.5,17 L11.5,17 L15,15.5 Q20,16 20,20Z" fill="%s"/>`, robe)
		b.WriteString(`<line x1="10" y1="17" x2="10" y2="20" stroke="#faf6ef" stroke-width="0.4"/>`)
		b.WriteString(`<circle cx="10" cy="18" r="0.3" fill="#faf6ef"/>`)
		b.WriteString(`<circle cx="10" cy="19.2" r="0.3" fill="#faf6ef"/>`)
	case 4: // Draped shawl with clasp.
		fmt.Fprintf(b, `<path d="M0,20 Q0,16 5,15.5 L8,17 L12,17 L15,15.5 Q20,16 20,20Z" fill="%s"/>`, robe)
		fmt.Fprintf(b, `<path d="M5,15.5 Q8,17.5 10,16" fill="none" stroke="%s" stroke-width="0.8"/>`, robe)
		fmt.Fprintf(b, `<circle cx="10" cy="16" r="0.6" fill="%s"/>`, gold)
	}
}

// writeFace draws the head/face shape.
func writeFace(b *strings.Builder, idx int, color string) {
	switch idx {
	case 0:
		fmt.Fprintf(b, `<circle cx="10" cy="11.5" r="4.5" fill="%s"/>`, color)
	case 1:
		fmt.Fprintf(b, `<ellipse cx="10" cy="11.5" rx="4" ry="4.8" fill="%s"/>`, color)
	case 2:
		fmt.Fprintf(b, `<ellipse cx="10" cy="11.5" rx="5" ry="4" fill="%s"/>`, color)
	case 3:
		fmt.Fprintf(b, `<rect x="5.5" y="7" width="9" height="9" rx="2.5" fill="%s"/>`, color)
	}
}

// writeEyes draws the eye style onto the face.
func writeEyes(b *strings.Builder, idx int) {
	switch idx {
	case 0: // Simple dots.
		fmt.Fprintf(b, `<circle cx="8" cy="10.5" r="0.7" fill="%s"/>`, dark)
		fmt.Fprintf(b, `<circle cx="12" cy="10.5" r="0.7" fill="%s"/>`, dark)
	case 1: // Narrow squint.
		fmt.Fprintf(b, `<ellipse cx="8" cy="10.5" rx="0.9" ry="0.4" fill="%s"/>`, dark)
		fmt.Fprintf(b, `<ellipse cx="12" cy="10.5" rx="0.9" ry="0.4" fill="%s"/>`, dark)
	case 2: // Wide with pupils.
		b.WriteString(`<circle cx="8" cy="10.5" r="1.1" fill="white"/>`)
		fmt.Fprintf(b, `<circle cx="8.2" cy="10.5" r="0.6" fill="%s"/>`, dark)
		b.WriteString(`<circle cx="12" cy="10.5" r="1.1" fill="white"/>`)
		fmt.Fprintf(b, `<circle cx="12.2" cy="10.5" r="0.6" fill="%s"/>`, dark)
	case 3: // Stern lines.
		fmt.Fprintf(b, `<line x1="7" y1="10.5" x2="9" y2="10.5" stroke="%s" stroke-width="0.8" stroke-linecap="round"/>`, dark)
		fmt.Fprintf(b, `<line x1="11" y1="10.5" x2="13" y2="10.5" stroke="%s" stroke-width="0.8" stroke-linecap="round"/>`, dark)
	case 4: // Red.
		b.WriteString(`<circle cx="8" cy="10.5" r="1.2" fill="#FF3C00"/>`)
		b.WriteString(`<circle cx="12" cy="10.5" r="1.2" fill="#FF3C00"/>`)
		b.WriteString(`<circle cx="8" cy="10.5" r="0.35" fill="#161612"/>`)
		b.WriteString(`<circle cx="12" cy="10.5" r="0.35" fill="#161612"/>`)
	}
}

// writeMouth draws the mouth expression.
func writeMouth(b *strings.Builder, idx int) {
	switch idx {
	case 0: // Neutral.
		fmt.Fprintf(b, `<line x1="8.5" y1="13.5" x2="11.5" y2="13.5" stroke="%s" stroke-width="0.5" stroke-linecap="round"/>`, dark)
	case 1: // Smirk.
		fmt.Fprintf(b, `<path d="M8.5 13.5Q10 13.5 11.5 13" stroke="%s" stroke-width="0.5" fill="none" stroke-linecap="round"/>`, dark)
	case 2: // Surprised.
		fmt.Fprintf(b, `<ellipse cx="10" cy="13.5" rx="1" ry="0.8" fill="%s"/>`, dark)
	case 3: // Grimace with teeth.
		fmt.Fprintf(b, `<line x1="8.5" y1="13.5" x2="11.5" y2="13.5" stroke="%s" stroke-width="0.8" stroke-linecap="round"/>`, dark)
		b.WriteString(`<line x1="9.5" y1="13.1" x2="9.5" y2="13.9" stroke="white" stroke-width="0.3"/>`)
		b.WriteString(`<line x1="10.5" y1="13.1" x2="10.5" y2="13.9" stroke="white" stroke-width="0.3"/>`)
	case 4: // Wavy/uneasy.
		fmt.Fprintf(b, `<path d="M8.5 13.5Q9.2 13 10 13.5Q10.8 14 11.5 13.5" stroke="%s" stroke-width="0.5" fill="none" stroke-linecap="round"/>`, dark)
	}
}

// writeHair draws the hair, adjusted for hat coverage.
func writeHair(b *strings.Builder, style int, color string, hat int) {
	// A hood covers all hair.
	if hat == hatHood {
		return
	}
	showTop := hat == hatCirclet

	switch style {
	case 0: // Short sides.
		fmt.Fprintf(b, `<rect x="5" y="8.5" width="1.5" height="4" rx="0.7" fill="%s"/>`, color)
		fmt.Fprintf(b, `<rect x="13.5" y="8.5" width="1.5" height="4" rx="0.7" fill="%s"/>`, color)
	case 1: // Medium with top.
		fmt.Fprintf(b, `<rect x="5" y="8" width="1.5" height="5" rx="0.7" fill="%s"/>`, color)
		fmt.Fprintf(b, `<rect x="13.5" y="8" width="1.5" height="5" rx="0.7" fill="%s"/>`, color)
		if showTop {
			fmt.Fprintf(b, `<ellipse cx="10" cy="7" rx="4.5" ry="1.5" fill="%s"/>`, color)
		}
	case 2: // Long flowing.
		fmt.Fprintf(b, `<rect x="4.5" y="8" width="2" height="7" rx="1" fill="%s"/>`, color)
		fmt.Fprintf(b, `<rect x="13.5" y="8" width="2" height="7" rx="1" fill="%s"/>`, color)
	case 3: // Wild spiky.
		fmt.Fprintf(b, `<rect x="5" y="8.5" width="1.3" height="3.5" rx="0.5" fill="%s"/>`, color)
		fmt.Fprintf(b, `<rect x="13.7" y="8.5" width="1.3" height="3.5" rx="0.5" fill="%s"/>`, color)
		if showTop {
			fmt.Fprintf(b, `<polygon points="6,8 5,4.5 7.5,7" fill="%s"/>`, color)
			fmt.Fprintf(b, `<polygon points="9,7 8,3.5 10.5,6" fill="%s"/>`, color)
			fmt.Fprintf(b, `<polygon points="11,7 12,3.5 9.5,6" fill="%s"/>`, color)
			fmt.Fprintf(b, `<polygon points="14,8 15,4.5 12.5,7" fill="%s"/>`, color)
		}
	}
}

// writeFacialHair draws optional facial hair. Category selects the type
// (none, full beard, beard only, beard+mustache, mustache only) and style
// picks a variant within that type. Mustaches sit above the mouth (y~12),
// beards sit below it (y>=14.5), and full beards cover the lower face
// (the mouth renders on top).
func writeFacialHair(b *strings.Builder, category, style int, color string) {
	switch category {
	case 0: // None.
	case 1: // Full beard.
		writeFullBeard(b, style, color)
	case 2: // Beard only.
		writeBeardOnly(b, style, color)
	case 3: // Beard and mustache.
		writeBeardOnly(b, style, color)
		writeMustache(b, style, color)
	case 4: // Mustache only.
		writeMustache(b, style, color)
	}
}

// writeFullBeard draws a large beard covering the lower face.
func writeFullBeard(b *strings.Builder, style int, color string) {
	switch style {
	case 0: // Round full beard.
		fmt.Fprintf(b, `<path d="M6 12.5Q6 17 10 17.5Q14 17 14 12.5" fill="%s"/>`, color)
	case 1: // Long flowing beard.
		fmt.Fprintf(b, `<path d="M6.5 12.5Q6 19 10 19.5Q14 19 13.5 12.5" fill="%s"/>`, color)
	case 2: // Tapered beard.
		fmt.Fprintf(b, `<path d="M6.5 12.5Q7 17 10 18Q13 17 13.5 12.5" fill="%s"/>`, color)
	}
}

// writeBeardOnly draws a chin/jaw beard below the mouth.
func writeBeardOnly(b *strings.Builder, style int, color string) {
	switch style {
	case 0: // Short rounded chin beard.
		fmt.Fprintf(b, `<path d="M7.5 14.5Q10 17.5 12.5 14.5" fill="%s"/>`, color)
	case 1: // Long pointed beard.
		fmt.Fprintf(b, `<path d="M7 14.5Q10 20 13 14.5" fill="%s"/>`, color)
	case 2: // Wide square beard.
		fmt.Fprintf(b, `<rect x="7" y="14.5" width="6" height="3" rx="1" fill="%s"/>`, color)
	}
}

// writeMustache draws a mustache above the mouth.
func writeMustache(b *strings.Builder, style int, color string) {
	switch style {
	case 0: // Handlebar.
		fmt.Fprintf(b, `<path d="M7.5 12.3Q8.5 13 10 12.3Q11.5 13 12.5 12.3" fill="%s"/>`, color)
	case 1: // Thick chevron.
		fmt.Fprintf(b, `<path d="M7.5 12.8L10 12L12.5 12.8Q10 12.3 7.5 12.8Z" fill="%s"/>`, color)
	case 2: // Thin pencil.
		fmt.Fprintf(b, `<line x1="7.8" y1="12.5" x2="10" y2="12.2" stroke="%s" stroke-width="0.5" stroke-linecap="round"/>`, color)
		fmt.Fprintf(b, `<line x1="10" y1="12.2" x2="12.2" y2="12.5" stroke="%s" stroke-width="0.5" stroke-linecap="round"/>`, color)
	}
}

// writeHat draws the headwear.
func writeHat(b *strings.Builder, style int, color string) {
	switch style {
	case hatWizard: // Crooked pointed wizard hat with crescent bend.
		fmt.Fprintf(b, `<path d="M5.5,8 Q5.5,3 7.5,0 Q14,1.5 14.5,8Z" fill="%s"/>`, color)
		fmt.Fprintf(b, `<rect x="4" y="7" width="12" height="1.5" rx="0.5" fill="%s"/>`, color)
		fmt.Fprintf(b, `<circle cx="7.5" cy="1" r="0.8" fill="%s"/>`, gold)
	case hatHood: // Hood front edge.
		fmt.Fprintf(b, `<path d="M4 12Q4 5 10 3Q16 5 16 12" fill="none" stroke="%s" stroke-width="1"/>`, color)
	case hatWideBrim: // Wide-brim hat.
		fmt.Fprintf(b, `<rect x="6" y="2" width="8" height="6" rx="3" fill="%s"/>`, color)
		fmt.Fprintf(b, `<ellipse cx="10" cy="7.5" rx="8" ry="1.5" fill="%s"/>`, color)
		fmt.Fprintf(b, `<ellipse cx="10" cy="7.5" rx="8" ry="1.5" fill="none" stroke="%s" stroke-width="0.4"/>`, dark)
	case hatCirclet: // Golden circlet with gem.
		fmt.Fprintf(b, `<rect x="5.5" y="7" width="9" height="1.2" rx="0.5" fill="%s"/>`, gold)
		fmt.Fprintf(b, `<circle cx="10" cy="7.5" r="0.8" fill="%s"/>`, color)
	case hatTurban: // Turban.
		fmt.Fprintf(b, `<ellipse cx="10" cy="6" rx="5" ry="3.5" fill="%s"/>`, color)
		fmt.Fprintf(b, `<circle cx="10" cy="3.5" r="1" fill="%s"/>`, gold)
	}
}

// writeStaff draws a wizard staff behind the character.
func writeStaff(b *strings.Builder) {
	b.WriteString(`<line x1="2.5" y1="4" x2="2.5" y2="19" stroke="#8b6040" stroke-width="1" stroke-linecap="round"/>`)
	fmt.Fprintf(b, `<circle cx="2.5" cy="3.5" r="1.5" fill="%s" opacity="0.8"/>`, gold)
}

// writeGadget draws the accessory (except glasses, which are drawn last).
func writeGadget(b *strings.Builder, gadget int, _ string) {
	switch gadget {
	case gadgetNone, gadgetStaff, gadgetGlasses:
		// Staff is drawn earlier; glasses are drawn later.
	case gadgetTarotCard:
		b.WriteString(`<rect x="15" y="12" width="3.5" height="5" rx="0.5" fill="#faf6ef" stroke="#d4c4a8" stroke-width="0.3"/>`)
		fmt.Fprintf(b, `<circle cx="16.75" cy="14.5" r="0.8" fill="%s"/>`, gold)
	case gadgetCrystal:
		b.WriteString(`<circle cx="16.5" cy="15" r="2" fill="#a0c0e0" opacity="0.7"/>`)
		b.WriteString(`<circle cx="15.8" cy="14.3" r="0.5" fill="white" opacity="0.6"/>`)
	case gadgetStar:
		fmt.Fprintf(b, `<polygon points="17,10 17.4,11.5 19,11.5 17.8,12.5 18.2,14 17,13 15.8,14 16.2,12.5 15,11.5 16.6,11.5" fill="%s"/>`, gold)
	case gadgetPendant:
		fmt.Fprintf(b, `<line x1="10" y1="15.5" x2="10" y2="17.5" stroke="%s" stroke-width="0.3"/>`, gold)
		fmt.Fprintf(b, `<circle cx="10" cy="18" r="1" fill="%s"/>`, gold)
	}
}

// writeGlasses draws round spectacles over the eyes.
func writeGlasses(b *strings.Builder) {
	b.WriteString(`<circle cx="8" cy="10.5" r="1.5" fill="none" stroke="#5a3a20" stroke-width="0.4"/>`)
	b.WriteString(`<circle cx="12" cy="10.5" r="1.5" fill="none" stroke="#5a3a20" stroke-width="0.4"/>`)
	b.WriteString(`<line x1="9.5" y1="10.5" x2="10.5" y2="10.5" stroke="#5a3a20" stroke-width="0.4"/>`)
}
