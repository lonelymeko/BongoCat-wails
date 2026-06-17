package services

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ImportedModel struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type mverConfig struct {
	Decoration mverDecoration `json:"decoration"`
	Standard   mverModeConfig `json:"standard"`
	Keyboard   mverModeConfig `json:"keyboard"`
	Gamepad    mverModeConfig `json:"gamepad"`
}

type mverDecoration struct {
	L2DCorrect        float64   `json:"l2d_correct"`
	L2DOffset         []float64 `json:"l2d_offset"`
	L2DHorizontalFlip bool      `json:"l2d_horizontal_flip"`
}

type mverModeConfig struct {
	Keyboard  [][]int `json:"keyboard"`
	Hand      [][]int `json:"hand"`
	LeftHand  [][]int `json:"lefthand"`
	RightHand [][]int `json:"righthand"`
}

type mverModeSpec struct {
	mode           string
	sourceDir      string
	backgroundName string
	leftDir        string
	handDir        string
	rightDir       string
	leftCombos     [][]int
	handCombos     [][]int
	rightCombos    [][]int
	layout         *modelLayout
}

type modelLayout struct {
	Scale      float64 `json:"scale,omitempty"`
	OffsetX    float64 `json:"offsetX,omitempty"`
	OffsetY    float64 `json:"offsetY,omitempty"`
	Mirror     bool    `json:"mirror,omitempty"`
	BehindBase bool    `json:"behindBase,omitempty"`
}

func importModel(from, to string) ([]ImportedModel, error) {
	from = filepath.Clean(from)

	if hasFile(from, "cat.model3.json") {
		if err := copyDir(from, to); err != nil {
			return nil, err
		}

		return []ImportedModel{{
			Path: to,
			Mode: detectBongoModelMode(to),
		}}, nil
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
	candidates := []string{path, filepath.Join(path, "bongo cat mver0.1.6")}

	if entries, err := os.ReadDir(path); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			candidates = append(candidates, filepath.Join(path, entry.Name()))
		}
	}

	for _, candidate := range candidates {
		if hasFile(candidate, "config.json") && hasFile(filepath.Join(candidate, "img", "standard", "cat_model"), "cat.model3.json") {
			return candidate, true
		}
	}

	return "", false
}

func importMverModel(root, to string) ([]ImportedModel, error) {
	config, err := readMverConfig(filepath.Join(root, "config.json"))
	if err != nil {
		return nil, err
	}

	specs := []mverModeSpec{
		{
			mode:           "standard",
			sourceDir:      filepath.Join(root, "img", "standard"),
			backgroundName: "mousebg.png",
			leftDir:        "keyboard",
			handDir:        "hand",
			leftCombos:     config.Standard.Keyboard,
			handCombos:     config.Standard.Hand,
			layout:         config.standardLayout(),
		},
		{
			mode:           "keyboard",
			sourceDir:      filepath.Join(root, "img", "keyboard"),
			backgroundName: "bg.png",
			leftDir:        "lefthand",
			rightDir:       "righthand",
			leftCombos:     config.Keyboard.LeftHand,
			rightCombos:    config.Keyboard.RightHand,
		},
		{
			mode:           "gamepad",
			sourceDir:      filepath.Join(root, "img", "gamepad"),
			backgroundName: "bg.png",
			leftDir:        "lefthand",
			rightDir:       "righthand",
			leftCombos:     config.Gamepad.LeftHand,
			rightCombos:    config.Gamepad.RightHand,
		},
	}

	imported := make([]ImportedModel, 0, len(specs))
	for _, spec := range specs {
		if !hasFile(filepath.Join(spec.sourceDir, "cat_model"), "cat.model3.json") {
			continue
		}

		modePath := filepath.Join(to, spec.mode)
		if err := convertMverMode(spec, modePath); err != nil {
			return nil, err
		}

		imported = append(imported, ImportedModel{
			Path: modePath,
			Mode: spec.mode,
		})
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
	err = json.Unmarshal(data, &config)
	return config, err
}

func (c mverConfig) standardLayout() *modelLayout {
	scale := c.Decoration.L2DCorrect
	if scale == 0 {
		scale = 1
	}

	layout := &modelLayout{
		Scale:      scale,
		Mirror:     c.Decoration.L2DHorizontalFlip,
		BehindBase: true,
	}

	if len(c.Decoration.L2DOffset) > 0 {
		layout.OffsetX = c.Decoration.L2DOffset[0]
	}
	if len(c.Decoration.L2DOffset) > 1 {
		layout.OffsetY = c.Decoration.L2DOffset[1]
	}

	return layout
}

func convertMverMode(spec mverModeSpec, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}

	if err := copyDir(filepath.Join(spec.sourceDir, "cat_model"), dst); err != nil {
		return err
	}

	resourcesDir := filepath.Join(dst, "resources")
	if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
		return err
	}

	background := filepath.Join(spec.sourceDir, spec.backgroundName)
	if !hasPath(background) {
		background = firstExistingPath(
			filepath.Join(spec.sourceDir, "mousebg.png"),
			filepath.Join(spec.sourceDir, "tabletbg.png"),
			filepath.Join(spec.sourceDir, "bg.png"),
		)
	}
	if background == "" {
		return fmt.Errorf("missing background image for %s", spec.mode)
	}

	if err := copyFile(background, filepath.Join(resourcesDir, "background.png")); err != nil {
		return err
	}
	if err := copyFile(background, filepath.Join(resourcesDir, "cover.png")); err != nil {
		return err
	}
	if spec.layout != nil {
		data, err := json.Marshal(spec.layout)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(resourcesDir, "layout.json"), data, 0o644); err != nil {
			return err
		}
	}

	if err := copyMverKeyLayers(
		filepath.Join(spec.sourceDir, spec.leftDir),
		filepath.Join(spec.sourceDir, spec.handDir),
		filepath.Join(resourcesDir, "left-keys"),
		spec.leftCombos,
		spec.handCombos,
		spec.mode == "gamepad",
		false,
	); err != nil {
		return err
	}
	if spec.rightDir != "" {
		if err := copyMverKeyLayers(
			filepath.Join(spec.sourceDir, spec.rightDir),
			"",
			filepath.Join(resourcesDir, "right-keys"),
			spec.rightCombos,
			nil,
			spec.mode == "gamepad",
			true,
		); err != nil {
			return err
		}
	}

	return nil
}

func copyMverKeyLayers(srcDir, handDir, dstDir string, combos, handCombos [][]int, gamepad bool, right bool) error {
	if len(combos) == 0 || !hasPath(srcDir) {
		return nil
	}

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	for index, combo := range combos {
		src := filepath.Join(srcDir, strconv.Itoa(index)+".png")
		if !hasPath(src) {
			continue
		}

		name := comboName(combo, gamepad, right, index)
		if name == "" {
			continue
		}

		dst := filepath.Join(dstDir, name+".png")
		handSrc := matchingHandLayer(handDir, handCombos, combo, index)
		if handSrc != "" {
			if err := compositePNG(dst, src, handSrc); err != nil {
				return err
			}
			continue
		}

		if err := copyFile(src, dst); err != nil {
			return err
		}
	}

	return nil
}

func matchingHandLayer(handDir string, handCombos [][]int, combo []int, fallbackIndex int) string {
	if handDir == "" || !hasPath(handDir) {
		return ""
	}

	index := fallbackIndex
	for i, handCombo := range handCombos {
		if intSliceEqual(handCombo, combo) {
			index = i
			break
		}
	}

	path := filepath.Join(handDir, strconv.Itoa(index)+".png")
	if hasPath(path) {
		return path
	}

	return ""
}

func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func compositePNG(dst string, layers ...string) error {
	var canvas *image.RGBA

	for _, layer := range layers {
		img, err := decodePNG(layer)
		if err != nil {
			return err
		}

		bounds := img.Bounds()
		if canvas == nil {
			canvas = image.NewRGBA(bounds)
		}

		draw.Draw(canvas, bounds, img, bounds.Min, draw.Over)
	}

	if canvas == nil {
		return fmt.Errorf("no layers to composite")
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	return png.Encode(out, canvas)
}

func decodePNG(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return png.Decode(file)
}

func comboName(combo []int, gamepad bool, right bool, index int) string {
	if gamepad {
		if right {
			names := []string{"South", "East", "West", "North", "RightTrigger", "RightTrigger2"}
			if index < len(names) {
				return names[index]
			}
		} else {
			names := []string{"DPadUp", "DPadRight", "DPadDown", "DPadLeft", "LeftTrigger", "LeftTrigger2"}
			if index < len(names) {
				return names[index]
			}
		}
	}

	for i := len(combo) - 1; i >= 0; i-- {
		if name := windowsVirtualKeyName(combo[i]); name != "" {
			return name
		}
	}

	return ""
}

func windowsVirtualKeyName(code int) string {
	switch {
	case code >= 65 && code <= 90:
		return "Key" + string(rune(code))
	case code >= 48 && code <= 57:
		return "Num" + string(rune(code))
	case code >= 96 && code <= 105:
		return "Num" + strconv.Itoa(code-96)
	}

	switch code {
	case 1:
		return "Left"
	case 2:
		return "Right"
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

func firstExistingPath(paths ...string) string {
	for _, path := range paths {
		if hasPath(path) {
			return path
		}
	}
	return ""
}

func hasFile(dir, name string) bool {
	return hasPath(filepath.Join(dir, name))
}

func hasPath(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
