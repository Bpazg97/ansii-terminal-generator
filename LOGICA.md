# Lógica del editor — documento de referencia

> Documento actualizado. Refleja el estado actual del código.

---

## 1. Tipos de dato fundamentales

### `Color` (`canvas.go`)
```
-1  → ColorDefault (color del terminal, sin fuerza)
0-7  → colores estándar ANSI
8-15 → colores brillantes
16-255 → cubo 6×6×6 + rampa de grises (ANSI 256)
```
`ColorDefault` (-1) en FG o BG significa "no aplicar color; usa el del terminal".

### `Cell` (`canvas.go`)
```go
Cell { Char rune, FG Color, BG Color }
```
- **Celda vacía canónica**: `BlankCell = {Char: ' ', FG: -1, BG: -1}`
- **Valor cero Go**: `{0, 0, 0}` — también tratado como vacío via `IsBlank()`
- `IsBlank()` devuelve `true` para ambos casos

### `Canvas`
Rejilla 2D de celdas. Dimensiones fijas en creación; `Resize()` existe pero no está expuesto en la UI.

---

## 2. Flujo principal (Bubbletea)

```
Terminal input
    │
    ▼
Update(msg)
    ├─ installConfirm activo → handleInstallConfirmKey
    ├─ saveMode activo       → handleSaveKey
    ├─ importMode activo     → handleImportKey
    ├─ showHelp activo       → cierra help
    └─ handleKey(msg)
           ├─ FocusCanvas → handleCanvasKey
           │       └─ textMode activo → handleTextKey
           ├─ FocusFGColor / FocusBGColor → handleColorKey
           └─ FocusChars → handleCharKey
                │
                ▼
           View() → renderiza título + canvas + sidebar + status
```

**Estados de modo exclusivo** (solo uno activo a la vez):
- `installConfirm` — confirmación antes de instalar splash
- `saveMode` — prompt de nombre de archivo
- `importMode` — prompt de ruta de importación + autocompletado
- `showHelp` — pantalla de ayuda
- `textMode` — dentro del canvas, escritura libre
- Foco de panel — `FocusCanvas`, `FocusFGColor`, `FocusBGColor`, `FocusChars`

---

## 3. Formatos de archivo

### `.ansii` — proyecto (JSON)
Solo almacena celdas no-vacías (`!cell.IsBlank()`). Carga y guarda colores exactos.
```json
{
  "version": 1, "width": 60, "height": 30,
  "cells": [
    { "x": 5, "y": 3, "char": "█", "fg": 11, "bg": -1 }
  ]
}
```
- `fg`/`bg`: `-1` = default, `0-255` = índice ANSI.

### `.ansi` — exportación (escape codes raw)
Archivo de texto con secuencias de escape ANSI. Para mostrarse con `cat`.
- Empieza con `\033[0m` para limpiar el estado previo del terminal.
- Por fila: solo escribe hasta la última celda no-vacía (`lastNonBlank`).
- Al final de cada fila no-vacía: `\033[0m\n`.
- Las filas completamente vacías se escriben como `\n` sola.
- `prevFG/prevBG` se inicializan a `ColorDefault` (no `-2`) para evitar códigos redundantes tras reset.

---

## 4. Lógica de guardado

### `[s]` / `Ctrl+S` — Guardar
1. Abre `saveMode` con `saveInput` pre-relleno con `m.filename` (o `"art.ansii"`).
2. El usuario puede cambiar la extensión para cambiar el formato.
3. Al confirmar, **detecta formato por extensión**:
   - `.ansi` → `exportANSI()` — no actualiza `m.filename`, no limpia `m.modified`
   - cualquier otra → `saveCanvas()` — actualiza `m.filename`, limpia `m.modified`
4. En el prompt, `Tab`/`Shift+Tab` activa el autocompletado de rutas desde `~/`.

### `[i]` — Instalar splash
1. Muestra `installConfirm` con aviso de riesgos: qué fichero se escribe, qué RC files se modifican.
2. `[Enter/y]` confirma: exporta a `changeExt(m.filename, ".ansi")` (o `"art.ansi"`), luego llama `installToShell()`.
3. `[Esc/n]` cancela sin modificar nada.

`installToShell()` copia el `.ansi` a `~/.config/ansii/splash.ansi` e inyecta/actualiza el bloque `# ansii-splash` en `.bashrc` y `.zshrc`.

---

## 5. Lógica de importación

### `[r]` — Texto plano
- Lee UTF-8, normaliza `\r\n`, crea canvas con el tamaño del contenido.
- Todas las celdas usan `FG: ColorDefault, BG: ColorDefault`.
- Tabs se saltan.

### `[g]` — Half-block (imagen → ANSI)
Algoritmo por cada celda del canvas:
1. Mapea la celda a dos regiones verticales de píxeles (upper/lower).
2. Promedia el color de cada región (`sampleRegion`).
3. Determina opacidad: alpha < 128 → transparente. Además, si `lumBG(r,g,b, isJPEG)` → transparente:
   - JPEG: `lum > 200` (más agresivo, sin canal alpha)
   - Otros: `lum > 230` (conservador)
4. Si ambas transparentes → celda vacía.
5. Si solo upper → `▀` con FG = color upper.
6. Si solo lower → `▄` con FG = color lower.
7. Si ambas opacas → `▀` con FG = upper, BG = lower.
8. Color → ANSI-256 con distancia "redmean" perceptual.

### `[a]` — ASCII art (imagen → caracteres)
1. `scaleX = imgW / targetW`, `scaleY = scaleX × 2` (aspect ratio 2:1 del terminal).
2. Promedia región de píxeles.
3. Si alpha < 128 → celda vacía.
4. Si `lumBG(r,g,b, isJPEG)` → celda vacía (elimina fondo).
5. Calcula luminancia BT.601: `(299R + 587G + 114B) / 1000`.
6. Mapea al ramp: `$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\|()1{}[]?-_+~<>i!lI;:,"^'. `
7. Asigna `FG = nearestANSI256(r, g, b)`, `BG = ColorDefault`.

---

## 6. Autocompletado de rutas (Tab en prompts)

- Input vacío o `~` → lista el home del usuario con prefijo `~/`.
- `~/ruta` → expande y lista.
- Directorios aparecen con `/` al final → siguiente Tab entra dentro.
- Archivos ocultos (`.nombre`) nunca se muestran.
- Máximo 64 resultados.
- `Tab` → siguiente, `Shift+Tab` → anterior.
- Cualquier tecla alfanumérica limpia la lista y recalcula en el próximo Tab.

---

## 7. Sistema de colores

### Paleta de 256 colores (panel interactivo)
- Grid 16×16: `Color = row * 16 + col` → colores 0-255.
- 8 filas visibles a la vez; scroll automático al navegar con ↑↓.
- Al entrar al panel (Tab/f/b), el cursor **salta al color FG o BG actual**.
- `colorScrollY` se ajusta automáticamente para mantener el cursor visible.

### `ColorDefault` (-1)
En el export se emite como `\033[39m` (FG default) o `\033[49m` (BG default).

---

## 8. Celda vacía

`IsBlank()` devuelve `true` para:
- `{Char: ' ', FG: -1, BG: -1}` — celda blank canónica (`BlankCell`)
- `{Char: 0, FG: 0, BG: 0}` — valor cero Go (nunca se crea explícitamente)

`saveCanvas`: omite celdas donde `IsBlank()` es true.
`lastNonBlank`: incluye celda si `!IsBlank()`.
`exportANSI`: `cell.Char == 0` se reemplaza por `' '` antes de emitir.

---

## 9. Navegación de paneles

```
Canvas ──[Tab]──→ FGColor ──[Tab]──→ Characters ──[Tab]──→ Canvas
Canvas ←──[Shift+Tab]── Characters ←──[Shift+Tab]── BGColor ←──[Shift+Tab]── Canvas
```

Dentro de FGColor/BGColor:
- `[f]` → FGColor (cursor salta al color FG actual)
- `[b]` → BGColor (cursor salta al color BG actual)
- `[d]` → reset a default y vuelve al canvas
- `Space` → aplica y se queda en el panel
- `Enter` → aplica y vuelve al canvas
- `Esc` → **cancela** (vuelve sin aplicar)

---

## 10. Viewport / scroll

```
canvasViewSize():
  w = termW - 36 (sidebar) - 2 (borde)
  h = termH - 6 (título + borde + label + status)
```

Fuera del área del canvas: se muestra `·` en gris oscuro.

---

## 11. Modos de texto (`[t]`)

- En modo texto, **cualquier tecla imprimible dibuja** el carácter.
- `Enter`: baja una fila, vuelve a la columna inicial del modo.
- `Esc` / `q`: sale del modo texto.
- `Ctrl+S`: abre el prompt de guardado.

---

## 12. CLI flags

| Flag | Descripción |
|------|-------------|
| `-f <file>` | Abre o crea un proyecto `.ansii` |
| `-w`, `-h` | Dimensiones del canvas (solo archivo nuevo) |
| `-import <file>` | Importa `.txt` como canvas |
| `-img <image>` | Importa imagen como half-block |
| `-ascii <image>` | Importa imagen como ASCII art |
| `-imgw <n>` | Anchura objetivo para importación de imagen |
| `-show <file>` | Muestra un `.ansi` en el terminal y sale |
| `-install` | Re-inyecta el splash en `.bashrc`/`.zshrc` |
| `-install -f my.ansii` | Carga el proyecto, exporta a `.ansi` e instala |
