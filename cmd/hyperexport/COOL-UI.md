# Cool UI Enhancement 🎨

The interactive TUI has been upgraded with a **cyberpunk/neon aesthetic** for an ultra-modern, visually stunning experience!

## 🌟 What's New

### 1. Neon/Cyberpunk Color Scheme
```
Neon Cyan:    #00FFFF  (Primary actions, headers)
Neon Magenta: #FF00FF  (Borders, accents)
Neon Green:   #39FF14  (Success, selections)
Neon Blue:    #00D9FF  (Info, highlights)
Neon Purple:  #BF00FF  (Borders, decorations)
Neon Orange:  #FF6600  (Warnings, hints)
Dark BG:      #0a0e27  (Background)
Light BG:     #1a1e37  (Highlights)
```

### 2. ASCII Art Banner
```
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║   ╦ ╦╦ ╦╔═╗╔═╗╦═╗  ╔═╗═╗ ╦╔═╗╔═╗╦═╗╔╦╗                 ║
║   ╠═╣╚╦╝╠═╝║╣ ╠╦╝  ║╣ ╔╩╦╝╠═╝║ ║╠╦╝ ║                  ║
║   ╩ ╩ ╩ ╩  ╚═╝╩╚═  ╚═╝╩ ╚═╩  ╚═╝╩╚═ ╩                  ║
║                                                          ║
║        ⚡ Multi-Cloud VM Export Platform ⚡             ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
```

### 3. Modern Status Bar
```
╭──────────────────────────────────────────────────────────────────────────╮
│ 📊 201 Total │ 👁  201 Visible │ ✓ 3 Selected │ 🔍 ubuntu │ ⚡ on       │
╰──────────────────────────────────────────────────────────────────────────╯
```

Features:
- Color-coded sections
- Rounded borders with neon purple
- Dynamic width
- Real-time updates

### 4. Contextual Hint Messages

**No Selection:**
```
╭──────────────────────────────────────────────────────────────╮
│ 💡 Press SPACE to select VMs │ A to select all │ ENTER to continue │
╰──────────────────────────────────────────────────────────────╯
```

**Single Selection:**
```
╭───────────────────────────────────────────────────────────────╮
│ ✓ 1 VM selected │ Select more with SPACE │ 200 VMs available │
╰───────────────────────────────────────────────────────────────╯
```

**Multiple Selections:**
```
╭═══════════════════════════════════════════════════════════╮
║ 🚀 3 VMs ready for export │ Press ENTER to continue       ║
╰═══════════════════════════════════════════════════════════╯
```

Note: Changes borders (single → double) when ready to export!

### 5. Enhanced VM List

```
╭────────────────────────────────────────────────────────────────────────╮
│                                                                        │
│    ☐ ○ vm-dev-01                          4C  8G    100GB            │
│    ☐ ○ vm-dev-02                          2C  4G     50GB            │
│  ▶ ☑ ⚡ vm-prod-database                   8C 16G    500GB            │
│    ☐ ⚡ vm-prod-web                        4C  8G    200GB            │
│    ☑ ⚡ vm-prod-app                        8C 32G    1.0TB            │
│    ☐ ○ vm-test-01                         2C  4G     50GB            │
│                                                                        │
╰────────────────────────────────────────────────────────────────────────╯
```

**Visual Elements:**
- `☐` / `☑` - Checkbox states (gray → neon green)
- `○` - Powered off (gray)
- `⚡` - Powered on (neon green)
- `▶` - Current cursor (neon cyan)
- Background highlight on current row
- Color-coded resources:
  - CPU: Neon Cyan
  - Memory: Neon Magenta
  - Storage: Neon Orange

### 6. Dynamic Styling

**Unselected VM:**
- Name: Neon Blue
- Resources: Color-coded
- Power: Gray or Green

**Selected VM:**
- Name: **Bold Neon Green**
- Checkbox: **☑ Neon Green**
- Stands out from list

**Cursor Position:**
- Background: Light BG (#1a1e37)
- Cursor Icon: **▶** Neon Cyan
- Full row highlight
- Bold + underline

## 🎨 Color-Coded Information

| Element | Color | Purpose |
|---------|-------|---------|
| Headers | Neon Cyan | Primary branding |
| Borders | Neon Purple/Magenta | Visual separation |
| Success | Neon Green | Confirmations, selections |
| Info | Neon Blue | VM names, information |
| Warning | Neon Orange | Hints, storage |
| CPU | Neon Cyan | Resource info |
| Memory | Neon Magenta | Resource info |
| Power On | Neon Green | Status |
| Power Off | Muted Gray | Status |

## 🚀 Visual Comparison

### Before (Standard UI):
```
hyperexport - interactive vm export

📊 Total: 201 | Visible: 201 | ✅ Selected: 0

  [ ] esx8.0-ubuntu22.04         🟢  4C   8G    200GB
> [ ] esx8.0-win10-x86_64        🔴  8C  16G    500GB
  [ ] esx8.0-rhel9.5             🟢  8C  16G    500GB
```

### After (Cool Neon UI):
```
╔══════════════════════════════════════════════════════════╗
║   ╦ ╦╦ ╦╔═╗╔═╗╦═╗  ╔═╗═╗ ╦╔═╗╔═╗╦═╗╔╦╗                 ║
║   ╠═╣╚╦╝╠═╝║╣ ╠╦╝  ║╣ ╔╩╦╝╠═╝║ ║╠╦╝ ║                  ║
║   ╩ ╩ ╩ ╩  ╚═╝╩╚═  ╚═╝╩ ╚═╩  ╚═╝╩╚═ ╩                  ║
║        ⚡ Multi-Cloud VM Export Platform ⚡             ║
╚══════════════════════════════════════════════════════════╝

╭──────────────────────────────────────────────────────────╮
│ 📊 201 Total │ 👁  201 Visible │ ✓ 0 Selected            │
╰──────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────╮
│ 💡 Press SPACE to select VMs │ A to select all │ ENTER   │
╰──────────────────────────────────────────────────────────╯

╭─────────────────────────────────────────────────────────╮
│    ☐ ⚡ esx8.0-ubuntu22.04           4C  8G    200GB    │
│  ▶ ☐ ○ esx8.0-win10-x86_64          8C 16G    500GB    │
│    ☐ ⚡ esx8.0-rhel9.5               8C 16G    500GB    │
╰─────────────────────────────────────────────────────────╯
```

## 🎯 Design Philosophy

### Inspiration
- **Cyberpunk aesthetic** - Neon colors, dark backgrounds
- **Retro-futuristic** - ASCII art, box-drawing characters
- **Modern terminals** - Support for 256+ colors
- **Gaming UIs** - Status bars, resource indicators

### Principles
1. **High contrast** - Easy to read in all lighting
2. **Color meaning** - Each color has semantic purpose
3. **Visual hierarchy** - Important info stands out
4. **Responsive feedback** - Immediate visual response
5. **Aesthetic pleasure** - Looks cool while being functional

## 📖 Usage

```bash
# Launch the cool interactive TUI
hyperexport --interactive

# Or use the short form
hyperexport -tui
```

## 🎮 Interactivity

All visual elements respond to user actions:

- **Hover** (cursor position) - Background changes
- **Select** - Checkbox glows green
- **Power state** - Icon animates
- **Status** - Colors update
- **Borders** - Change when ready

## 🌈 Terminal Compatibility

Works best with:
- **iTerm2** (macOS)
- **Windows Terminal** (Windows 11)
- **GNOME Terminal** (Linux)
- **Konsole** (KDE)
- **Alacritty**
- **Kitty**
- **WezTerm**

Requires:
- 256-color support
- Unicode support
- True color recommended

## ⚡ Performance

All visual enhancements are:
- **Zero performance impact** on selections
- **Efficient rendering** with lipgloss
- **Minimal memory** overhead
- **Instant updates** on actions

## 🎨 Customization

Colors can be adjusted in `interactive_tui.go`:
```go
neonCyan      = lipgloss.Color("#00FFFF")  // Change primary color
neonMagenta   = lipgloss.Color("#FF00FF")  // Change accent
neonGreen     = lipgloss.Color("#39FF14")  // Change success
// ... etc
```

## 🚀 Future Enhancements

Planned improvements:
- [ ] Animated transitions
- [ ] Progress bar pulses
- [ ] Cursor animations
- [ ] Sound effects (optional)
- [ ] Theme presets (cyberpunk, matrix, retro)
- [ ] Custom ASCII art
- [ ] Particle effects
- [ ] More border styles

## 💡 Tips

1. **Maximize your terminal** - The UI looks best full-screen
2. **Use dark terminal theme** - Neon colors pop on dark backgrounds
3. **Enable true color** - For best gradient effects
4. **Adjust font size** - 12-14pt recommended

## 🎉 Showcase

The cool UI makes terminal work feel like:
- A retro arcade game
- Hacking in a cyberpunk movie
- Using a futuristic console
- Playing a text-based RPG

**It's not just a tool, it's an experience!** ⚡🎮🌟
