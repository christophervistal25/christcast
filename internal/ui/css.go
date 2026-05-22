//go:build gtk

package ui

const styleCSS = `
window.cct-overlay {
    background-color: #1c1c1e;
    border: 1px solid #3a3a3c;
    border-radius: 10px;
}
entry.cct-entry {
    background-color: transparent;
    color: #f5f5f7;
    border: none;
    box-shadow: none;
    padding: 14px 16px;
    font-size: 18px;
    caret-color: #f5f5f7;
}
entry.cct-entry:focus {
    outline: none;
    box-shadow: none;
}
separator.cct-sep {
    background-color: #2c2c2e;
    min-height: 1px;
}
list.cct-list {
    background-color: transparent;
    color: #f5f5f7;
}
list.cct-list row {
    padding: 8px 16px;
    border-radius: 6px;
    margin: 1px 6px;
}
list.cct-list row:selected {
    background-color: #0a84ff;
    color: white;
}
label.cct-base {
    font-size: 14px;
    font-weight: 500;
}
label.cct-path {
    font-size: 11px;
    color: #8e8e93;
}
list.cct-list row:selected label.cct-path {
    color: #d4e7ff;
}
label.cct-empty {
    color: #8e8e93;
    font-size: 13px;
    padding: 24px;
}
`
