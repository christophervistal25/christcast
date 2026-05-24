//go:build gtk

package ui

const styleCSS = `
/* Outer window is transparent so rounded corners reveal the desktop
   instead of the compositor-default black square. */
window.cct-overlay {
    background-color: transparent;
    font-family: -apple-system, "SF Pro Display", "Inter", "Segoe UI", "Cantarell", sans-serif;
    color: #f5f5f7;
}
/* Inner content box paints the actual rounded surface. */
window.cct-overlay > box {
    background-color: #1c1c1e;
    border: 1px solid #3a3a3c;
    border-radius: 10px;
}
window.cct-overlay decoration {
    background-color: transparent;
    box-shadow: none;
    border: none;
}

entry.cct-entry {
    background-color: transparent;
    background-image: none;
    color: #f5f5f7;
    border: none;
    border-image: none;
    box-shadow: none;
    outline: none;
    padding: 14px 16px;
    min-height: 48px;
    font-size: 17px;
    font-weight: 400;
    caret-color: #f5f5f7;
    font-family: -apple-system, "SF Pro Display", "Inter", "Segoe UI", "Cantarell", sans-serif;
}
entry.cct-entry:focus {
    outline: none;
    box-shadow: none;
    border: none;
}
entry.cct-entry selection {
    background-color: #0a84ff;
    color: #ffffff;
}
entry.cct-entry placeholder {
    color: #8e8e93;
}
entry.cct-entry image {
    color: #8e8e93;
}

separator.cct-sep {
    background-color: #2c2c2e;
    min-height: 1px;
    margin: 0;
    padding: 0;
    border: none;
}

list.cct-list {
    background-color: transparent;
    color: #f5f5f7;
    padding: 6px 6px;
}
list.cct-list row {
    padding: 8px 12px;
    min-height: 52px;
    border-radius: 8px;
    margin: 1px 0;
    background-color: transparent;
    color: #f5f5f7;
}
list.cct-list row:hover {
    background-color: #2c2c2e;
}
list.cct-list row:selected,
list.cct-list row:selected:focus {
    background-color: #0a84ff;
    color: #ffffff;
}

label.cct-base {
    font-size: 15px;
    font-weight: 500;
    color: #f5f5f7;
}
label.cct-path {
    font-size: 12px;
    font-weight: 400;
    color: #8e8e93;
}
list.cct-list row:selected label.cct-base {
    color: #ffffff;
}
list.cct-list row:selected label.cct-path {
    color: #ffffff;
}

label.cct-empty {
    color: #8e8e93;
    font-size: 13px;
    padding: 24px;
}

box.cct-hint-bar,
.cct-hint-bar {
    background-color: #2c2c2e;
    border-top: 1px solid #3a3a3c;
    min-height: 36px;
    padding: 6px 12px;
}

label.cct-hint,
.cct-hint {
    color: #8e8e93;
    font-size: 12px;
    margin: 0 6px;
}
label.cct-hint .key,
.cct-hint .key {
    font-family: "SF Mono", "JetBrains Mono", "Menlo", "Consolas", monospace;
    color: #f5f5f7;
    background-color: #3a3a3c;
    padding: 1px 6px;
    border-radius: 4px;
    margin-right: 4px;
    font-size: 11px;
}
`
