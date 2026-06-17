package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ImportedModel is one model produced by an import (a BongoCat model, or one
// mode of a legacy Bongo-Cat-Mver package).
type ImportedModel struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

// ---- legacy Bongo-Cat-Mver config (config.json) -----------------------------

type mverConfig struct {
	Decoration mverDecoration `json:"decoration"`
	Standard   mverModeConfig `json:"standard"`
	Keyboard   mverModeConfig `json:"keyboard"`
	Gamepad    mverModeConfig `json:"gamepad"`
}

type mverDecoration struct {
	L2DCorrect        float64   `json:"l2d_correct"`
	L2DHorizontalFlip bool      `json:"l2d_horizontal_flip"`
	LeftHanded        bool      `json:"leftHanded"`
	WindowSize        []int     `json:"window_size"`
	OffsetX           []int     `json:"offsetX"`
	OffsetY           []int     `json:"offsetY"`
	Scalar            []float64 `json:"scalar"`
	HandOffset        []int     `json:"hand_offset"`
	ArmLineColor      []int     `json:"armLineColor"`
}

type mverModeConfig struct {
	Keyboard  [][]int `json:"keyboard"`
	Hand      [][]int `json:"hand"`
	LeftHand  [][]int `json:"lefthand"`
	RightHand [][]int `json:"righthand"`
	Face      [][]int `json:"face"`
}

// ---- mver.json manifest (consumed by the frontend MverStage) ----------------
//
// Instead of lossily compositing Mver's full-frame sprite layers into BongoCat
// keycap slots, we now PRESERVE Mver's images verbatim and emit a manifest that
// lets the frontend replicate Mver's own draw logic (drawkeyboard = stack all
// matching combos; drawhand = most-recently-pressed combo wins; standard mode
// adds a procedural bezier arm + mouse-following device). See src/utils/mver.ts.

type mverManifest struct {
	Renderer   string                 `json:"renderer"` // always "mver"
	Mode       string                 `json:"mode"`     // standard | keyboard | gamepad
	WindowSize []int                  `json:"windowSize"`
	CatModel   string                 `json:"catModel,omitempty"`   // dir holding cat.model3.json (standard)
	Background string                 `json:"background,omitempty"` // relative png
	Decoration mverManifestDecoration `json:"decoration"`
	Device     *mverManifestDevice    `json:"device,omitempty"` // standard mouse/arm layer
	Layers     map[string][]mverLayer `json:"layers"`           // category -> combos
	Idle       map[string]string      `json:"idle,omitempty"`   // category -> idle image
}

type mverManifestDecoration struct {
	L2DCorrect float64 `json:"l2dCorrect"`
	L2DFlip    bool    `json:"l2dFlip"`
	LeftHanded bool    `json:"leftHanded"`
	OffsetX    int     `json:"offsetX"`
	OffsetY    int     `json:"offsetY"`
	Scale      float64 `json:"scale"`
	HandOffset []int   `json:"handOffset"`
	ArmColor   []int   `json:"armColor"`
}

type mverManifestDevice struct {
	Base  string `json:"base,omitempty"`
	Left  string `json:"left,omitempty"`
	Right string `json:"right,omitempty"`
	Side  string `json:"side,omitempty"`
	Arm   string `json:"arm,omitempty"`
	Up    string `json:"up,omitempty"`
}

type mverLayer struct {
	Keys []string `json:"keys"` // rdev-style key names (matches device-changed events)
	Img  string   `json:"img"`  // relative image path, e.g. "keyboard/0.png"
}

// ---- entry point ------------------------------------------------------------

func importModel(from, to string) ([]ImportedModel, error) {
	from = filepath.Clean(from)

	// Already a BongoCat model: copy as-is.
	if hasFile(from, "cat.model3.json") {
		if err := copyDir(from, to); err != nil {
			return nil, err
		}
		return []ImportedModel{{Path: to, Mode: detectBongoModelMode(to)}}, nil
	}

	mverRoot, ok := findMverRoot(from)
	if !ok {
		return nil, fmt.Errorf("unsupported model directory: %s", from)
	}

	return importMverModel(mverRoot, to)
}

func detectBongoModelMode(path string) string {
	files, err := os.ReadDir(filepath.Join(path, "resources", "right-keys"))
	if err != nil {
		return "standard"
	}
	for _, file := range files {
		name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		if name == "East" {
			return "gamepad"
		}
	}
	if len(files) > 0 {
		return "keyboard"
	}
	return "standard"
}

func findMverRoot(path string) (string, bool) {
	candidates := []string{path}
	if entries, err := os.ReadDir(path); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				candidates = append(candidates, filepath.Join(path, entry.Name()))
			}
		}
	}
	for _, candidate := range candidates {
		if hasFile(candidate, "config.json") &&
			hasFile(filepath.Join(candidate, "img", "standard", "cat_model"), "cat.model3.json") {
			return candidate, true
		}
	}
	return "", false
}

// ---- mver import ------------------------------------------------------------

// modeCategories returns the per-mode category->combos map and draw metadata.
type modeSpec struct {
	mode       string
	categories map[string][][]int // category name -> combos
	mouseIndex int                // which decoration offset/scalar slot to use
}

func importMverModel(root, to string) ([]ImportedModel, error) {
	config, err := readMverConfig(filepath.Join(root, "config.json"))
	if err != nil {
		return nil, err
	}

	specs := []modeSpec{
		{
			mode:       "standard",
			mouseIndex: 0,
			categories: map[string][][]int{
				"keyboard": config.Standard.Keyboard,
				"hand":     config.Standard.Hand,
				"face":     config.Standard.Face,
			},
		},
		{
			mode:       "keyboard",
			mouseIndex: 1,
			categories: map[string][][]int{
				"keyboard":  config.Keyboard.Keyboard,
				"lefthand":  config.Keyboard.LeftHand,
				"righthand": config.Keyboard.RightHand,
				"face":      config.Keyboard.Face,
			},
		},
		{
			mode:       "gamepad",
			mouseIndex: 1,
			categories: map[string][][]int{
				"keyboard":  config.Gamepad.Keyboard,
				"lefthand":  config.Gamepad.LeftHand,
				"righthand": config.Gamepad.RightHand,
				"face":      config.Gamepad.Face,
			},
		},
	}

	imported := make([]ImportedModel, 0, len(specs))
	for _, spec := range specs {
		sourceDir := filepath.Join(root, "img", spec.mode)
		if !hasPath(sourceDir) {
			continue
		}

		modePath := filepath.Join(to, spec.mode)
		if err := emitMverMode(config, spec, sourceDir, modePath); err != nil {
			return nil, err
		}
		imported = append(imported, ImportedModel{Path: modePath, Mode: spec.mode})
	}

	if len(imported) == 0 {
		return nil, fmt.Errorf("no importable Bongo Cat Mver modes found in %s", root)
	}
	return imported, nil
}

func readMverConfig(path string) (mverConfig, error) {
	var config mverConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	return config, json.Unmarshal(data, &config)
}

// emitMverMode converts one Mver mode. Modern Mver models are Live2D-centric:
// the visible character is the Cubism `cat_model` (the cat/avatar), animated via
// the model's own motions/expressions — NOT the 2D cat-paw sprites. For those we
// emit a plain BongoCat-compatible Live2D model so the existing (tested) Live2D
// renderer draws the character. Only genuinely sprite-based modes (no cat_model)
// fall back to the full-frame MverStage + mver.json.
func emitMverMode(config mverConfig, spec modeSpec, sourceDir, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}

	if hasFile(filepath.Join(sourceDir, "cat_model"), "cat.model3.json") {
		return emitLive2DModel(config, sourceDir, dst)
	}

	// Pure 2D sprite mode: preserve images verbatim and drive them via mver.json.
	if err := copyDir(sourceDir, dst); err != nil {
		return err
	}
	manifest := buildManifest(config, spec, dst)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, "mver.json"), data, 0o644)
}

type bongoLayout struct {
	Scale      float64 `json:"scale,omitempty"`
	Mirror     bool    `json:"mirror,omitempty"`
	BehindBase bool    `json:"behindBase,omitempty"`
}

// emitLive2DModel writes a BongoCat-compatible model: the Cubism model files at
// the root, plus resources/{background,cover,layout}. No keycaps, so no wrong
// overlays — just the character + the desk/device background.
func emitLive2DModel(config mverConfig, sourceDir, dst string) error {
	// Cubism model (cat.model3.json + moc3 + textures + physics/pose) at root.
	if err := copyDir(filepath.Join(sourceDir, "cat_model"), dst); err != nil {
		return err
	}

	resources := filepath.Join(dst, "resources")
	if err := os.MkdirAll(resources, 0o755); err != nil {
		return err
	}

	// Background (the desk / tablet / controller the character sits at).
	bg := firstExistingAbs(sourceDir,
		"l2dmousebg.png", "l2dtabletbg.png", "mousebg.png", "tabletbg.png", "bg.png")
	if bg != "" {
		if err := copyFile(bg, filepath.Join(resources, "background.png")); err != nil {
			return err
		}
	}

	// Card cover: prefer the model's cat.png preview, else the background.
	cover := filepath.Join(sourceDir, "cat.png")
	if !hasPath(cover) {
		cover = bg
	}
	if cover != "" {
		if err := copyFile(cover, filepath.Join(resources, "cover.png")); err != nil {
			return err
		}
	}

	layout := bongoLayout{
		Scale:  orDefault(config.Decoration.L2DCorrect, 1),
		Mirror: config.Decoration.L2DHorizontalFlip,
		// Keep the background (the tablet/desk, which includes the keyboard) at
		// the BOTTOM so the character's Live2D hands render on top of the keys.
		BehindBase: false,
	}
	data, err := json.MarshalIndent(layout, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(resources, "layout.json"), data, 0o644)
}

func firstExistingAbs(dir string, names ...string) string {
	for _, name := range names {
		p := filepath.Join(dir, name)
		if hasPath(p) {
			return p
		}
	}
	return ""
}

func buildManifest(config mverConfig, spec modeSpec, dst string) mverManifest {
	dec := config.Decoration

	manifest := mverManifest{
		Renderer:   "mver",
		Mode:       spec.mode,
		WindowSize: dec.WindowSize,
		Layers:     map[string][]mverLayer{},
		Idle:       map[string]string{},
		Decoration: mverManifestDecoration{
			L2DCorrect: orDefault(dec.L2DCorrect, 1),
			L2DFlip:    dec.L2DHorizontalFlip,
			LeftHanded: dec.LeftHanded,
			OffsetX:    nthInt(dec.OffsetX, spec.mouseIndex),
			OffsetY:    nthInt(dec.OffsetY, spec.mouseIndex),
			Scale:      orDefault(nthFloat(dec.Scalar, spec.mouseIndex), 1),
			HandOffset: dec.HandOffset,
			ArmColor:   dec.ArmLineColor,
		},
	}

	// Only standard mode renders the Live2D body; keyboard/gamepad are 2D only.
	if spec.mode == "standard" && hasFile(filepath.Join(dst, "cat_model"), "cat.model3.json") {
		manifest.CatModel = "cat_model"
	}

	// Background: prefer the live2d-specific backgrounds, then plain ones.
	manifest.Background = firstExistingRel(dst,
		"l2dmousebg.png", "l2dtabletbg.png", "mousebg.png", "tabletbg.png", "bg.png")

	// Build the per-category combo layers, mapping each combo's key codes to the
	// rdev-style names emitted by DeviceService, paired with category/<i>.png.
	for category, combos := range spec.categories {
		layers := make([]mverLayer, 0, len(combos))
		for i, combo := range combos {
			img := filepath.Join(category, strconv.Itoa(i)+".png")
			if !hasPath(filepath.Join(dst, img)) {
				continue
			}
			layers = append(layers, mverLayer{
				Keys: comboKeyNames(combo),
				Img:  filepath.ToSlash(img),
			})
		}
		if len(layers) > 0 {
			manifest.Layers[category] = layers
		}
	}

	// Idle (key-up) images for the hand layers.
	addIdle(manifest.Idle, dst, "hand", "up.png")
	addIdle(manifest.Idle, dst, "lefthand", "lefthand/leftup.png")
	addIdle(manifest.Idle, dst, "righthand", "righthand/rightup.png")

	// Standard mode draws a procedural arm + mouse-following device sprites.
	if spec.mode == "standard" {
		manifest.Device = &mverManifestDevice{
			Base:  relIfExists(dst, "mouse.png"),
			Left:  relIfExists(dst, "mouse_left.png"),
			Right: relIfExists(dst, "mouse_right.png"),
			Side:  relIfExists(dst, "mouse_side.png"),
			Arm:   relIfExists(dst, "arm.png"),
			Up:    relIfExists(dst, "up.png"),
		}
	}

	return manifest
}

// comboKeyNames maps a Bongo-Cat-Mver key combo (Windows virtual-key codes) to
// the rdev-style names the frontend matches against. Modifier codes map to the
// base name (e.g. "Shift"); the frontend normalises "ShiftLeft"/"ShiftRight"
// down to "Shift" so the match still works.
func comboKeyNames(combo []int) []string {
	names := make([]string, 0, len(combo))
	for _, code := range combo {
		if name := windowsVirtualKeyName(code); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func windowsVirtualKeyName(code int) string {
	switch {
	case code >= 65 && code <= 90: // A-Z
		return "Key" + string(rune(code))
	case code >= 48 && code <= 57: // 0-9
		return "Num" + string(rune(code))
	case code >= 96 && code <= 105: // numpad 0-9
		return "Num" + strconv.Itoa(code-96)
	case code >= 112 && code <= 123: // F1-F12
		return "F" + strconv.Itoa(code-111)
	}

	switch code {
	case 1:
		return "Left"
	case 2:
		return "Right"
	case 4:
		return "Middle"
	case 8:
		return "Backspace"
	case 9:
		return "Tab"
	case 13:
		return "Return"
	case 16:
		return "Shift"
	case 17:
		return "Control"
	case 18:
		return "Alt"
	case 20:
		return "CapsLock"
	case 27:
		return "Escape"
	case 32:
		return "Space"
	case 37:
		return "LeftArrow"
	case 38:
		return "UpArrow"
	case 39:
		return "RightArrow"
	case 40:
		return "DownArrow"
	case 46:
		return "Delete"
	case 91, 92:
		return "Meta"
	case 186:
		return "SemiColon"
	case 187:
		return "Equal"
	case 188:
		return "Comma"
	case 189:
		return "Minus"
	case 190:
		return "Dot"
	case 191:
		return "Slash"
	case 192:
		return "BackQuote"
	case 219:
		return "LeftBracket"
	case 220:
		return "BackSlash"
	case 221:
		return "RightBracket"
	case 222:
		return "Quote"
	default:
		return ""
	}
}

// ---- small helpers ----------------------------------------------------------

func addIdle(idle map[string]string, dst, category, rel string) {
	if hasPath(filepath.Join(dst, filepath.FromSlash(rel))) {
		idle[category] = rel
	}
}

func relIfExists(dst, rel string) string {
	if hasPath(filepath.Join(dst, filepath.FromSlash(rel))) {
		return rel
	}
	return ""
}

func firstExistingRel(dst string, rels ...string) string {
	for _, rel := range rels {
		if hasPath(filepath.Join(dst, filepath.FromSlash(rel))) {
			return rel
		}
	}
	return ""
}

func orDefault(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

func nthInt(s []int, i int) int {
	if i >= 0 && i < len(s) {
		return s[i]
	}
	if len(s) > 0 {
		return s[0]
	}
	return 0
}

func nthFloat(s []float64, i int) float64 {
	if i >= 0 && i < len(s) {
		return s[i]
	}
	if len(s) > 0 {
		return s[0]
	}
	return 0
}

func hasFile(dir, name string) bool {
	return hasPath(filepath.Join(dir, name))
}

func hasPath(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
